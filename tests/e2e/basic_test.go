package e2e

import (
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	oadpv1alpha1 "github.com/openshift/oadp-operator/api/v1alpha1"
)

var _ = Describe("OADP CLI Basic Tests", func() {
	Context("CLI Help Commands", func() {
		It("should display help when called with --help", func() {
			output := expectCLISuccess(cliBinaryPath, "--help")
			outputStr := string(output)

			Expect(outputStr).To(ContainSubstring("OADP CLI commands"))
			Expect(outputStr).To(ContainSubstring("Available Commands:"))
			Expect(outputStr).To(ContainSubstring("version"))
			Expect(outputStr).To(ContainSubstring("backup"))
		})

		It("should display version information", func() {
			output := expectCLISuccess(cliBinaryPath, "version")
			outputStr := string(output)

			// Should contain version information
			Expect(outputStr).To(ContainSubstring("Version:"))
		})
	})

	Context("DPA Configuration Tests", func() {
		It("should have a DPA configured in the cluster", func() {
			var dpa oadpv1alpha1.DataProtectionApplication
			err := k8sClient.Get(ctx, client.ObjectKey{
				Name:      dpaName,
				Namespace: testNamespace,
			}, &dpa)

			Expect(err).NotTo(HaveOccurred())
			Expect(dpa.Name).To(Equal(dpaName))
			Expect(dpa.Namespace).To(Equal(testNamespace))

			// Verify basic DPA configuration
			Expect(dpa.Spec.Configuration).NotTo(BeNil())
			Expect(dpa.Spec.Configuration.Velero).NotTo(BeNil())
			Expect(dpa.Spec.Configuration.Velero.DefaultPlugins).To(ContainElement(oadpv1alpha1.DefaultPluginAWS))
		})

		It("should have OADP operator running", func() {
			// This test verifies the OADP operator is running (already validated in prerequisites)
			Eventually(func() error {
				var deployments appsv1.DeploymentList
				err := k8sClient.List(ctx, &deployments, client.InNamespace(testNamespace))
				if err != nil {
					return err
				}

				for _, deployment := range deployments.Items {
					if deployment.Name == "openshift-adp-controller-manager" {
						if deployment.Status.ReadyReplicas > 0 {
							return nil
						}
					}
				}
				return fmt.Errorf("OADP operator not ready")
			}, 30*time.Second, 5*time.Second).Should(Succeed())
		})
	})

	Context("Basic CLI Commands", func() {
		It("should run backup command without arguments", func() {
			// This should show help for backup command
			output := expectCLISuccess(cliBinaryPath, "backup")
			outputStr := string(output)

			// Should contain backup-related help
			Expect(outputStr).To(ContainSubstring("backup"))
		})

		It("should handle invalid commands gracefully", func() {
			// Test that CLI handles invalid commands properly
			output, err := runCLICommand(cliBinaryPath, "invalid-command")

			// Should fail but not crash
			Expect(err).To(HaveOccurred())

			// Should contain error message
			outputStr := string(output)
			Expect(outputStr).To(ContainSubstring("unknown command"))
		})

		It("should run version command multiple times consistently", func() {
			// Run version command multiple times to ensure consistency
			var outputs []string
			for i := 0; i < 3; i++ {
				output := expectCLISuccess(cliBinaryPath, "version")
				outputs = append(outputs, string(output))
			}

			// All outputs should be identical
			for i := 1; i < len(outputs); i++ {
				Expect(outputs[i]).To(Equal(outputs[0]))
			}
		})
	})

	Context("Cluster Connectivity", func() {
		It("should be able to connect to the cluster", func() {
			// Test that we can connect to the cluster
			_, err := clientset.CoreV1().Namespaces().Get(ctx, "default", metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should have access to the openshift-adp namespace", func() {
			// Test that we can access the openshift-adp namespace
			_, err := clientset.CoreV1().Namespaces().Get(ctx, testNamespace, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("DPA Status Verification", func() {
		It("should have DPA in a valid state", func() {
			// This test should check for DPA existence and status
			Eventually(func() error {
				var dpa oadpv1alpha1.DataProtectionApplication
				err := k8sClient.Get(ctx, client.ObjectKey{
					Name:      dpaName,
					Namespace: testNamespace,
				}, &dpa)
				if err != nil {
					return fmt.Errorf("DPA not found: %v", err)
				}

				// Check DPA status
				if dpa.Status.Conditions == nil || len(dpa.Status.Conditions) == 0 {
					return fmt.Errorf("DPA status not ready")
				}

				// Look for reconciled condition
				for _, condition := range dpa.Status.Conditions {
					if condition.Type == "Reconciled" && condition.Status == metav1.ConditionTrue {
						return nil
					}
				}

				return fmt.Errorf("DPA not reconciled yet")
			}, testTimeout, pollInterval).Should(Succeed())
		})

		It("should have valid backup locations configured", func() {
			var dpa oadpv1alpha1.DataProtectionApplication
			err := k8sClient.Get(ctx, client.ObjectKey{
				Name:      dpaName,
				Namespace: testNamespace,
			}, &dpa)

			Expect(err).NotTo(HaveOccurred())
			Expect(dpa.Spec.BackupLocations).NotTo(BeEmpty())
			Expect(dpa.Spec.BackupLocations[0].Velero.Provider).To(Equal("aws"))
			Expect(dpa.Spec.BackupLocations[0].Velero.ObjectStorage.Bucket).To(Equal(os.Getenv("OADP_BUCKET")))
			Expect(dpa.Spec.BackupLocations[0].Velero.Config["region"]).To(Equal(os.Getenv("VSL_REGION")))
		})

		It("should have valid snapshot locations configured", func() {
			var dpa oadpv1alpha1.DataProtectionApplication
			err := k8sClient.Get(ctx, client.ObjectKey{
				Name:      dpaName,
				Namespace: testNamespace,
			}, &dpa)

			Expect(err).NotTo(HaveOccurred())
			Expect(dpa.Spec.SnapshotLocations).NotTo(BeEmpty())
			Expect(dpa.Spec.SnapshotLocations[0].Velero.Provider).To(Equal("aws"))
			Expect(dpa.Spec.SnapshotLocations[0].Velero.Config["region"]).To(Equal(os.Getenv("VSL_REGION")))
		})
	})

	Context("CLI Integration", func() {
		It("should handle invalid commands gracefully", func() {
			// Test that invalid commands fail appropriately
			err := expectCLIFailure(cliBinaryPath, "invalid-command")
			Expect(err).To(HaveOccurred())
		})

		It("should handle invalid flags gracefully", func() {
			// Test that invalid flags are handled properly
			err := expectCLIFailure(cliBinaryPath, "version", "--invalid-flag")
			Expect(err).To(HaveOccurred())
		})
	})
})

var _ = Describe("OADP CLI Dummy Test", func() {
	It("should always pass as a dummy test", func() {
		By("Running a simple dummy test")
		// This is a simple test that should always pass
		// It's useful for testing the test framework itself
		Expect(true).To(BeTrue())
		By("Dummy test completed successfully")
	})

	It("should be able to run basic CLI commands", func() {
		By("Testing basic CLI help command")

		// Execute the CLI help command
		output := expectCLISuccess(cliBinaryPath, "--help")

		// Log the output for debugging
		By("=== CLI Help Output ===")
		By(string(output))
		By("=== End CLI Help Output ===")

		// Verify the output contains expected content
		Expect(string(output)).To(ContainSubstring("OADP CLI commands"))
		Expect(string(output)).To(ContainSubstring("Available Commands:"))

		By("CLI help command executed successfully")
	})
})
