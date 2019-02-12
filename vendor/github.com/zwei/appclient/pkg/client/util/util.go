package util

import "io/ioutil"

func ReadFile(pfile string) ([]byte , error) {
	data, err := ioutil.ReadFile(pfile)
	if err != nil {
		return nil, err
	}
	return data, nil
}
