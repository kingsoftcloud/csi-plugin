package ks3

import (
	"os"
	"testing"
)

func TestCreateCredentialFile(t *testing.T) {
	testCases := []struct {
		name              string
		bucket            string
		secrets           map[string]string
		expectFileContent string
		expectFileName    string
		expectError       bool
	}{
		{
			name:              "fakebucket passwd file is not exist and success",
			bucket:            "fakebucket",
			secrets:           map[string]string{"SecretId": "fakesid", "SecretKey": "fakeskey"},
			expectFileContent: "fakebucket:fakesid:fakeskey",
			expectFileName:    "/tmp/ks3_mnt/e9704d26-e64e-4821-8528-a6e6896700f1_fakebucket",
			expectError:       false,
		},
		{
			name:              "fakebucket passwd file is exist and success",
			bucket:            "fakebucket",
			secrets:           map[string]string{"SecretId": "fakesid", "SecretKey": "fakeskey"},
			expectFileContent: "fakebucket:fakesid:fakeskey",
			expectFileName:    "/tmp/ks3_mnt/e9704d26-e64e-4821-8528-a6e6896700f1_fakebucket",
			expectError:       false,
		},
		{
			name:              "fakebucket passwd file and write another bucket sid skey success",
			bucket:            "fakebucket2",
			secrets:           map[string]string{"SecretId": "fakesid2", "SecretKey": "fakeskey2"},
			expectFileContent: "fakebucket2:fakesid2:fakeskey2",
			expectFileName:    "/tmp/ks3_mnt/e9704d26-e64e-4821-8528-a6e6896700f1_fakebucket2",
			expectError:       false,
		},
		{
			name:              "fakebucket passwd file exist and sid skey have space charactor",
			bucket:            "fakebucket2",
			secrets:           map[string]string{"SecretId": "fakesid2 ", "SecretKey": "fakeskey2\n"},
			expectFileContent: "fakebucket2:fakesid2:fakeskey2",
			expectFileName:    "/tmp/ks3_mnt/e9704d26-e64e-4821-8528-a6e6896700f1_fakebucket2",
			expectError:       false,
		},
		{
			name:              "fakebucket passwd file exist and sid skey changed",
			bucket:            "fakebucket",
			secrets:           map[string]string{"SecretId": "fakesid22 ", "SecretKey": "fakeskey22\n"},
			expectFileContent: "fakebucket:fakesid22:fakeskey22",
			expectFileName:    "/tmp/ks3_mnt/e9704d26-e64e-4821-8528-a6e6896700f1_fakebucket",
			expectError:       false,
		},
		{
			name:              "secret is not valid fail",
			bucket:            "fakebucket23",
			secrets:           map[string]string{"SecretId111": "fakesid22", "SecretKey111": "fakeskey22"},
			expectFileContent: "fakebucket2:fakesid22:fakeskey22",
			expectFileName:    "/tmp/ks3_mnt/e9704d26-e64e-4821-8528-a6e6896700f1_fakebucket23",
			expectError:       true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filepath, err := createCredentialFile("testvol", tc.bucket, tc.secrets)
			if err != nil && tc.expectError {
				t.Fatalf("find error %v", err)
			}
			if err == nil {
				t.Log("error occur is in expected return")
				return
			}
			if _, err := os.Stat(filepath); err != nil {
				t.Fatalf("find error %v", err)
			}
			if filepath != tc.expectFileName {
				t.Fatalf("filepath %s is not equal expectFileName %s", filepath, tc.expectFileName)
			}
			data, err := os.ReadFile(filepath)
			if err != nil {
				t.Fatalf("read fakePassword file error %v", err)
			}
			if string(data) != tc.expectFileContent {
				t.Fatalf("file content %s is not equal expectFileContent %s", string(data), tc.expectFileContent)
			}
		})
	}
	if err := os.RemoveAll("/tmp/ks3_mnt/e9704d26-e64e-4821-8528-a6e6896700f1_fakebucket"); err != nil {
		t.Fatalf("remove password file error %v", err)
	}
	if err := os.Remove("/tmp/ks3_mnt/e9704d26-e64e-4821-8528-a6e6896700f1_fakebucket2"); err != nil {
		t.Fatalf("Remove password file error %v", err)
	}
	if err := os.Remove("/tmp/ks3_mnt/e9704d26-e64e-4821-8528-a6e6896700f1_fakebucket23"); err != nil {
		t.Fatalf("Remove password file error %v", err)
	}
}
