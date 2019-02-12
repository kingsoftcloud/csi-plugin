package types

import "fmt"

type AKSK struct {
	AK string
	SK string
}

func (k *AKSK) SetAKSK(ak, sk string) {
	k.AK = ak
	k.SK = sk
}

func (k *AKSK) GetAKSK() (string, string, error) {
	if len(k.AK) == 0 {
		return "", "",  fmt.Errorf("not found cluster ak")
	}
	if len(k.SK) == 0 {
		return "", "",  fmt.Errorf("not found cluster sk")
	}

	return k.AK, k.SK, nil
}