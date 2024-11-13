package securedload

import "errors"

func GetAssetSecured(name string) ([]byte, error) {
	var err error
	for _, v := range knownProtectedLoader {
		loadedData, errLoad := v.VerifyAndLoad(name)
		if errLoad == nil {
			return loadedData, nil
		}
		err = errors.New(" is not loading executable file")
	}
	return nil, err
}
