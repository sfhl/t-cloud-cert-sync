package main

import (
    "context"
    "crypto/sha256"
    "fmt"
    "log"
    "os"
    "os/signal"
    "sync"
    "syscall"
    "time"

    "github.com/opentelekomcloud/gophertelekomcloud"
    "github.com/opentelekomcloud/gophertelekomcloud/openstack"
    "github.com/opentelekomcloud/gophertelekomcloud/openstack/elb/v3/certificates"
    "github.com/opentelekomcloud/gophertelekomcloud/openstack/elb/v3/listeners"
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/types"
    "k8s.io/client-go/informers"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/rest"
    "k8s.io/client-go/tools/cache"
)

const (
    AnnotationListenerID  = "t-cloud.telekom.com/listener-id"
    AnnotationSyncedState = "t-cloud.telekom.com/synced-state" // NEW: persistent memory
)

// thread-safe cache for the certificate hashes/states
var certHashes sync.Map

func main() {
    log.Println("Starting T Cloud Public ELB Cert Sync Controller...")

    // 1. init K8s client
    k8sConfig, err := rest.InClusterConfig()
    if err != nil {
        log.Fatalf("Error loading k8s config: %v", err)
    }
    clientset, err := kubernetes.NewForConfig(k8sConfig)
    if err != nil {
        log.Fatalf("Error creating the k8s clients: %v", err)
    }

    // 2. init t cloud client (pure AK/SK, does no IAM-Token)
    provider, err := openstack.NewClient(os.Getenv("OS_AUTH_URL"))
    if err != nil {
        log.Fatalf("Error creating the base client: %v", err)
    }

    provider.AKSKAuthOptions = golangsdk.AKSKAuthOptions{
        AccessKey: os.Getenv("OS_ACCESS_KEY"),
        SecretKey: os.Getenv("OS_SECRET_KEY"),
        ProjectId: os.Getenv("OS_PROJECT_ID"),
    }

    // 3. setup ELB Service Client
    elbEndpoint := fmt.Sprintf("https://elb.%s.otc.t-systems.com/v3/%s/elb/", os.Getenv("OS_REGION_NAME"), os.Getenv("OS_PROJECT_ID"))
    elbClient := &golangsdk.ServiceClient{
        ProviderClient: provider,
        Endpoint:       elbEndpoint,
        MoreHeaders: map[string]string{
            "X-Project-Id": os.Getenv("OS_PROJECT_ID"),
        },
    }

    // 4. K8s informer setup (watches all namespaces)
    factory := informers.NewSharedInformerFactory(clientset, time.Hour*12)
    secretInformer := factory.Core().V1().Secrets().Informer()

    secretInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
        AddFunc: func(obj interface{}) {
            // WE PASS clientset HERE TO ALLOW PATCHING
            handleSecretChange(clientset, obj, elbClient)
        },
        UpdateFunc: func(oldObj, newObj interface{}) {
            // WE PASS clientset HERE TO ALLOW PATCHING
            handleSecretChange(clientset, newObj, elbClient)
        },
    })

    // 5. start controller
    stopCh := make(chan struct{})
    defer close(stopCh)

    go factory.Start(stopCh)

    if !cache.WaitForCacheSync(stopCh, secretInformer.HasSynced) {
        log.Fatal("Timed out waiting for caches to sync")
    }

    log.Println("Controller running and waiting for secret events...")

    // graceful shutdown
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh
    log.Println("Stopping controller...")
}

func handleSecretChange(k8sClient *kubernetes.Clientset, obj interface{}, elbClient *golangsdk.ServiceClient) {
    secret, ok := obj.(*corev1.Secret)
    if !ok {
        return
    }

    // check if annotations exists
    listenerID, hasAnnotation := secret.Annotations[AnnotationListenerID]
    if !hasAnnotation || listenerID == "" {
        return // secret is not marked for t cloud
    }

    certData := secret.Data["tls.crt"]
    keyData := secret.Data["tls.key"]

    if len(certData) == 0 || len(keyData) == 0 {
        log.Printf("Warning: secret %s/%s has annotation, but no tls data.", secret.Namespace, secret.Name)
        return
    }

    // calculcate unambiguous state (certificate hash + listener ID)
    cacheKey := fmt.Sprintf("%s/%s", secret.Namespace, secret.Name)
    currentState := fmt.Sprintf("%x-%s", sha256.Sum256(certData), listenerID)

    // --- 1. CHECK MEMORY CACHE (for fast runtime checks) ---
    lastState, stateExists := certHashes.Load(cacheKey)
    if stateExists && lastState == currentState {
        return
    }

    // --- 2. CHECK PERSISTENT STATE (survives pod restarts) ---
    if secret.Annotations[AnnotationSyncedState] == currentState {
        // Was already synced before a pod restart.
        // Just load into memory to avoid unnecessary API calls.
        certHashes.Store(cacheKey, currentState)
        return
    }

    log.Printf("Change in certificate in %s/%s recognized! processing listener %s...", secret.Namespace, secret.Name, listenerID)

    // 1. get current listener state to find OLD certificate
    var oldCertID string
    currentListener, err := listeners.Get(elbClient, listenerID).Extract()
    if err == nil && currentListener != nil {
        oldCertID = currentListener.DefaultTlsContainerRef
    } else {
        log.Printf("Warning: Unable to retrieve the current listener state (perhaps it doesn't exist?): %v", err)
    }

    // 2. create NEW certificate in t cloud
    certName := fmt.Sprintf("k8s-%s-%d", secret.Name, time.Now().Unix())
    certCreateOpts := certificates.CreateOpts{
        Name:        certName,
        Certificate: string(certData),
        PrivateKey:  string(keyData),
    }

    newCert, err := certificates.Create(elbClient, certCreateOpts).Extract()
    if err != nil {
        log.Printf("Error uploading the new certificate %s to T Cloud Public: %v", cacheKey, err)
        return
    }
    log.Printf("New certificate created in T Cloud Public. ID: %s", newCert.ID)

    // 3. bind NEW certificate to listener
    listenerUpdateOpts := listeners.UpdateOpts{
        DefaultTlsContainerRef: &newCert.ID,
    }

    _, err = listeners.Update(elbClient, listenerID, listenerUpdateOpts).Extract()
    if err != nil {
        log.Printf("Error binding to listener %s: %v", listenerID, err)
        // fallback: if bind fails, we directly try to cleanup the new certificate
        certificates.Delete(elbClient, newCert.ID)
        return
    }

    log.Printf("Certificate from %s successfully bound to listener %s!", cacheKey, listenerID)

    // --- 3. WRITE STATE BACK TO KUBERNETES SECRET (Persistent Memory) ---
    patchData := fmt.Sprintf(`{"metadata":{"annotations":{"%s":"%s"}}}`, AnnotationSyncedState, currentState)
    _, err = k8sClient.CoreV1().Secrets(secret.Namespace).Patch(context.TODO(), secret.Name, types.MergePatchType, []byte(patchData), metav1.PatchOptions{})
    if err != nil {
        log.Printf("Warning: Could not write sync state annotation to secret %s: %v", cacheKey, err)
    } else {
        log.Printf("Sync state successfully saved to K8s secret %s.", cacheKey)
    }

    // 4. cleanup OLD certificate (asynchron)
    if oldCertID != "" && oldCertID != newCert.ID {
        log.Printf("Starting Cleanup for old certificate: %s", oldCertID)

        // start cleanup within a go routine in the background,
        // so that the controller does not get stuck.
        go func(certToDelete string, client *golangsdk.ServiceClient) {
            // Short pause, since load balancers operate asynchronously.
            // If the certificate is deleted too quickly, T Cloud throws a "Resource in use" error.
            time.Sleep(5 * time.Second)

            err := certificates.Delete(client, certToDelete).ExtractErr()
            if err != nil {
                log.Printf("Cleanup-error: Unable to delete the old certificate %s from T Cloud Public: %v", certToDelete, err)
            } else {
                log.Printf("Cleanup: Old certificate %s successfully deleted!", certToDelete)
            }
        }(oldCertID, elbClient)
    }

    // 5. save current state in cache
    certHashes.Store(cacheKey, currentState)
}
