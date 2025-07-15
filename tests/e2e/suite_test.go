package e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	k8sscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	oadpv1alpha1 "github.com/openshift/oadp-operator/api/v1alpha1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
)

// Test configuration constants
const (
	testNamespace         = "openshift-adp"
	testTimeout           = 10 * time.Minute
	pollInterval          = 5 * time.Second
	dpaName               = "test-dpa"
	backupStorageLocation = "test-backup-location"
	// Required environment variables for DPA creation
	oadpCredFileEnv = "OADP_CRED_FILE"
	oadpBucketEnv   = "OADP_BUCKET"
	ciCredFileEnv   = "CI_CRED_FILE"
	vslRegionEnv    = "VSL_REGION"
)

var (
	ctx           context.Context
	k8sClient     client.Client
	clientset     kubernetes.Interface
	cliBinaryPath string
	scheme        *runtime.Scheme
)

// TestE2E runs the e2e test suite
func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OADP CLI E2E Suite")
}

var _ = BeforeSuite(func() {
	By("Setting up test environment")

	// Initialize Kubernetes client
	setupKubernetesClient()

	// Build CLI binary
	buildCLIBinary()

	// Validate prerequisites (including environment variables)
	validatePrerequisites()

	// Create and wait for DPA to be ready
	createBasicDPA()
	waitForDPAReady()

	By("Test environment setup complete")
})

var _ = AfterSuite(func() {
	By("Cleaning up test environment")

	// Clean up DPA first
	cleanupDPA()

	// Clean up cloud credentials secret
	cleanupCloudCredentialsSecret()

	// Clean up test resources
	cleanupTestResources()

	// Clean up binary
	cleanupBinary(cliBinaryPath)

	By("Test environment cleanup complete")
})

// setupKubernetesClient initializes the Kubernetes client
func setupKubernetesClient() {
	By("Setting up Kubernetes client")

	// Load kubeconfig
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		homeDir, err := os.UserHomeDir()
		Expect(err).NotTo(HaveOccurred())
		kubeconfig = filepath.Join(homeDir, ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	Expect(err).NotTo(HaveOccurred())

	// Create clientset
	clientset, err = kubernetes.NewForConfig(config)
	Expect(err).NotTo(HaveOccurred())

	// Create scheme with core Kubernetes, OADP and Velero types
	scheme = runtime.NewScheme()
	Expect(k8sscheme.AddToScheme(scheme)).To(Succeed())
	Expect(oadpv1alpha1.AddToScheme(scheme)).To(Succeed())
	Expect(velerov1.AddToScheme(scheme)).To(Succeed())

	// Create controller-runtime client
	k8sClient, err = client.New(config, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())

	// Initialize context
	ctx = context.Background()

	By("Kubernetes client setup complete")
}

// buildCLIBinary builds the CLI binary for testing
func buildCLIBinary() {
	By("Building CLI binary")

	// Use the existing build function from common.go
	cliBinaryPath = buildCLIBinaryFromProject()

	By(fmt.Sprintf("CLI binary built at: %s", cliBinaryPath))
}

// validatePrerequisites checks if all required components are available
func validatePrerequisites() {
	By("Validating prerequisites")

	// Check if we can connect to the cluster
	_, err := clientset.CoreV1().Namespaces().Get(ctx, "default", metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred(), "Failed to connect to Kubernetes cluster")

	// Check if openshift-adp namespace exists
	_, err = clientset.CoreV1().Namespaces().Get(ctx, testNamespace, metav1.GetOptions{})
	if err != nil {
		By("⚠️  openshift-adp namespace not found. Please install OADP operator first.")
		By("   kubectl create namespace openshift-adp")
		By("   kubectl apply -f https://github.com/openshift/oadp-operator/releases/latest/download/oadp-operator.yaml")
		Expect(err).NotTo(HaveOccurred(), "openshift-adp namespace not found. Please install OADP operator first.")
	}

	// Check if OADP operator is installed
	Eventually(func() error {
		var deployments appsv1.DeploymentList
		err := k8sClient.List(ctx, &deployments, client.InNamespace(testNamespace))
		if err != nil {
			return fmt.Errorf("failed to list deployments: %v", err)
		}

		for _, deployment := range deployments.Items {
			if deployment.Name == "openshift-adp-controller-manager" {
				if deployment.Status.ReadyReplicas > 0 {
					return nil
				}
				return fmt.Errorf("OADP operator found but not ready (ready replicas: %d)", deployment.Status.ReadyReplicas)
			}
		}
		return fmt.Errorf("OADP operator controller-manager deployment not found")
	}, 60*time.Second, 5*time.Second).Should(Succeed(), "OADP operator should be installed and running")

	// Validate required environment variables for DPA creation
	By("Validating required environment variables for DPA creation")

	requiredEnvVars := map[string]string{
		oadpCredFileEnv: "Path to AWS credentials file",
		oadpBucketEnv:   "S3 bucket name for backups",
		ciCredFileEnv:   "Path to CI credentials file",
		vslRegionEnv:    "AWS region for backup storage",
	}

	for envVar, description := range requiredEnvVars {
		value := os.Getenv(envVar)
		if value == "" {
			By(fmt.Sprintf("❌ Missing required environment variable: %s (%s)", envVar, description))
			By("Required environment variables:")
			By("  export OADP_CRED_FILE=<path-to-aws-credentials>")
			By("  export OADP_BUCKET=<your-s3-bucket-name>")
			By("  export CI_CRED_FILE=<path-to-ci-credentials>")
			By("  export VSL_REGION=<aws-region>")
			Expect(value).NotTo(BeEmpty(), fmt.Sprintf("Environment variable %s is required for DPA creation", envVar))
		} else {
			By(fmt.Sprintf("✅ %s = %s", envVar, value))
		}
	}

	// Validate that credential files exist
	credFile := os.Getenv(oadpCredFileEnv)
	if credFile != "" {
		// Expand ~ to home directory
		if strings.HasPrefix(credFile, "~/") {
			homeDir, err := os.UserHomeDir()
			Expect(err).NotTo(HaveOccurred())
			credFile = filepath.Join(homeDir, credFile[2:])
		}

		_, err := os.Stat(credFile)
		Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Credential file not found: %s", credFile))
		By(fmt.Sprintf("✅ Credential file exists: %s", credFile))
	}

	By("Prerequisites validation complete")
}

// createCloudCredentialsSecret creates the cloud-credentials secret for AWS
func createCloudCredentialsSecret() {
	By("Creating cloud-credentials secret")

	credFile := os.Getenv(oadpCredFileEnv)

	// Expand ~ to home directory
	if strings.HasPrefix(credFile, "~/") {
		homeDir, err := os.UserHomeDir()
		Expect(err).NotTo(HaveOccurred())
		credFile = filepath.Join(homeDir, credFile[2:])
	}

	// Read the credentials file
	credData, err := os.ReadFile(credFile)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to read credentials file: %s", credFile))

	// Create the secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cloud-credentials",
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			"cloud": credData,
		},
	}

	// Delete existing secret if it exists
	var existingSecret corev1.Secret
	err = k8sClient.Get(ctx, client.ObjectKey{
		Name:      "cloud-credentials",
		Namespace: testNamespace,
	}, &existingSecret)

	if err == nil {
		// Secret exists, delete it first
		err = k8sClient.Delete(ctx, &existingSecret)
		Expect(err).NotTo(HaveOccurred(), "Failed to delete existing cloud-credentials secret")

		// Wait for secret to be deleted
		Eventually(func() error {
			var secret corev1.Secret
			err := k8sClient.Get(ctx, client.ObjectKey{
				Name:      "cloud-credentials",
				Namespace: testNamespace,
			}, &secret)

			if err != nil && strings.Contains(err.Error(), "not found") {
				return nil
			}

			if err != nil {
				return err
			}

			return fmt.Errorf("secret still exists")
		}, 30*time.Second, 2*time.Second).Should(Succeed(), "Existing secret should be deleted")
	}

	// Create the new secret
	err = k8sClient.Create(ctx, secret)
	Expect(err).NotTo(HaveOccurred(), "Failed to create cloud-credentials secret")

	By("Cloud credentials secret created successfully")
}

// cleanupCloudCredentialsSecret removes the cloud-credentials secret
func cleanupCloudCredentialsSecret() {
	By("Cleaning up cloud-credentials secret")

	var secret corev1.Secret
	err := k8sClient.Get(ctx, client.ObjectKey{
		Name:      "cloud-credentials",
		Namespace: testNamespace,
	}, &secret)

	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			By("Secret not found, nothing to clean up")
			return
		}
		By(fmt.Sprintf("Error getting secret: %v", err))
		return
	}

	// Delete the secret
	err = k8sClient.Delete(ctx, &secret)
	if err != nil {
		By(fmt.Sprintf("Error deleting secret: %v", err))
		return
	}

	By("Cloud credentials secret cleanup complete")
}

// createBasicDPA creates a basic DataProtectionApplication for testing
func createBasicDPA() {
	By("Creating basic DPA configuration")

	// First create the cloud credentials secret
	createCloudCredentialsSecret()

	// Get environment variables
	credFile := os.Getenv(oadpCredFileEnv)
	bucket := os.Getenv(oadpBucketEnv)
	region := os.Getenv(vslRegionEnv)

	// Expand ~ to home directory for credential file
	if strings.HasPrefix(credFile, "~/") {
		homeDir, err := os.UserHomeDir()
		Expect(err).NotTo(HaveOccurred())
		credFile = filepath.Join(homeDir, credFile[2:])
	}

	// Create a minimal DPA configuration
	dpa := &oadpv1alpha1.DataProtectionApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dpaName,
			Namespace: testNamespace,
		},
		Spec: oadpv1alpha1.DataProtectionApplicationSpec{
			Configuration: &oadpv1alpha1.ApplicationConfig{
				Velero: &oadpv1alpha1.VeleroConfig{
					DefaultPlugins: []oadpv1alpha1.DefaultPlugin{
						oadpv1alpha1.DefaultPluginAWS,
					},
				},
			},
			BackupLocations: []oadpv1alpha1.BackupLocation{
				{
					Velero: &velerov1.BackupStorageLocationSpec{
						Provider: "aws",
						Default:  true,
						StorageType: velerov1.StorageType{
							ObjectStorage: &velerov1.ObjectStorageLocation{
								Bucket: bucket,
								Prefix: "velero",
							},
						},
						Config: map[string]string{
							"region": region,
						},
						Credential: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "cloud-credentials",
							},
							Key: "cloud",
						},
					},
				},
			},
			SnapshotLocations: []oadpv1alpha1.SnapshotLocation{
				{
					Velero: &velerov1.VolumeSnapshotLocationSpec{
						Provider: "aws",
						Config: map[string]string{
							"region": region,
						},
						Credential: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "cloud-credentials",
							},
							Key: "cloud",
						},
					},
				},
			},
		},
	}

	// Create the DPA
	err := k8sClient.Create(ctx, dpa)
	Expect(err).NotTo(HaveOccurred(), "Failed to create DPA")

	By("DPA created successfully")
}

// waitForDPAReady waits for the DPA to be ready
func waitForDPAReady() {
	By("Waiting for DPA to be ready")

	Eventually(func() error {
		var dpa oadpv1alpha1.DataProtectionApplication
		err := k8sClient.Get(ctx, client.ObjectKey{
			Name:      dpaName,
			Namespace: testNamespace,
		}, &dpa)
		if err != nil {
			return fmt.Errorf("failed to get DPA: %v", err)
		}

		// Check if DPA has conditions and is ready
		if dpa.Status.Conditions == nil || len(dpa.Status.Conditions) == 0 {
			return fmt.Errorf("DPA status conditions not available")
		}

		// Look for ready condition
		for _, condition := range dpa.Status.Conditions {
			if condition.Type == "Reconciled" && condition.Status == metav1.ConditionTrue {
				return nil
			}
		}

		return fmt.Errorf("DPA not ready yet")
	}, testTimeout, pollInterval).Should(Succeed(), "DPA should be ready")

	By("DPA is ready")
}

// cleanupDPA removes the test DPA
func cleanupDPA() {
	By("Cleaning up DPA")

	// Get the DPA
	var dpa oadpv1alpha1.DataProtectionApplication
	err := k8sClient.Get(ctx, client.ObjectKey{
		Name:      dpaName,
		Namespace: testNamespace,
	}, &dpa)

	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			By("DPA not found, nothing to clean up")
			return
		}
		By(fmt.Sprintf("Error getting DPA: %v", err))
		return
	}

	// Delete the DPA
	err = k8sClient.Delete(ctx, &dpa)
	if err != nil {
		By(fmt.Sprintf("Error deleting DPA: %v", err))
		return
	}

	// Wait for DPA to be deleted
	Eventually(func() error {
		var dpa oadpv1alpha1.DataProtectionApplication
		err := k8sClient.Get(ctx, client.ObjectKey{
			Name:      dpaName,
			Namespace: testNamespace,
		}, &dpa)

		if err != nil && strings.Contains(err.Error(), "not found") {
			return nil
		}

		if err != nil {
			return err
		}

		return fmt.Errorf("DPA still exists")
	}, 2*time.Minute, 5*time.Second).Should(Succeed(), "DPA should be deleted")

	By("DPA cleanup complete")
}

// cleanupTestResources removes any remaining test resources
func cleanupTestResources() {
	By("Cleaning up test resources")

	// Clean up any backups that may have been created during testing
	var backups velerov1.BackupList
	err := k8sClient.List(ctx, &backups, client.InNamespace(testNamespace))
	if err == nil {
		for _, backup := range backups.Items {
			if strings.HasPrefix(backup.Name, "test-") {
				err := k8sClient.Delete(ctx, &backup)
				if err != nil {
					fmt.Printf("Warning: Failed to delete backup %s: %v\n", backup.Name, err)
				}
			}
		}
	}

	By("Test resources cleanup complete")
}
