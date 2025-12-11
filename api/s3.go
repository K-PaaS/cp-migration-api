package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rclone/rclone/librclone/librclone"
	"kps-migration-api/config"
	"kps-migration-api/model"
	"net/http"
)

var (
	rcloneInitialize = librclone.Initialize
	rcloneRPC        = librclone.RPC
)

func defaultDecodeSyncConfig(payload model.HybridPayload) (model.SyncConfig, error) {
	return decodePayload[model.SyncConfig](payload)
}

func defaultDecodeStorageConfig(payload model.HybridPayload) (model.StorageConfig, error) {
	return decodePayload[model.StorageConfig](payload)
}

var (
	decodeSyncConfig    = defaultDecodeSyncConfig
	decodeStorageConfig = defaultDecodeStorageConfig
)

// @Summary Synchronize between storage
// @Description Synchronize between storage.
// @Description Example request body before encoding :
// @Description {
// @Description     "src": {
// @Description         "storageType": "s3",
// @Description         "endpoint": "http://url.co.kr",
// @Description         "accessKeyId": "admin",
// @Description         "secretAccessKey": "admin",
// @Description         "bucket": "abc"
// @Description     },
// @Description     "dst": {
// @Description         "storageType": "s3",
// @Description         "endpoint": "http://url.com",
// @Description         "accessKeyId": "admin",
// @Description         "secretAccessKey": "admin",
// @Description         "bucket": "abcd"
// @Description     }
// @Description }
// @Tags Migration
// @Accept json
// @Produce json
// @Param payload body model.HybridPayload true "encode base64 model.SyncConfig" SchemaExample({"key":"iCJcdWA198EF1uuzePq8M+Onsx8/4KJ9Mm9YDY+j+KUSxV+/4YyRkeDiuVAyRHkU8UUTyXS/zRhb2t8gUMKJiJaReLFgVyrweVMxRLk2Jx1yshNQ++Y/Cq4+Jvvk9lqsqpbmcoP2VrQoP4C4vymWQ+jwaV/vkJfUK0j6l6LlZenL8PmSzbhzO8kFVaS69TULu4IKbAK9YBeFN/QFhR84/St2fkd5RO8drgN0nPjieVlePc1N9mQewxkPQFXEsDAS3ws3xiBrw7YpHMn7mAk2VZ5y4F3vuEbGL7dIgWbG5UqGXdFGpyUnD91b7Tq+oMCrYhFIquOOIi3eqXirgmdZqOpmdBjDpM97t3oLKpCJDKTU9ceQSAVRSfP4HAN2LSbHTj/n4IZsVzpfcgTuXq0jHIRfrM3VPWnkidpI0v5cWMpJwAGvWcYmR7mwKCeSTJJ5/iE+0usbc/BaUQxFberLf2+QgIsNaR9V9iVnswwMQ0KVGGitsrEuCiTPv5Nlhi/WK8qvw42sFt098EPToWJ7/yHYAEEwt5Wr+4mAh3rNiLFfFB85JWM4ScrHyufHcHU4weqb/Bg5MD2ctO5fL7sp0vSOLSS3ziBpganU8Wzj7rLbaMrJbrhyZuhm6d7XdH/TvROFbTKO5bXHxwOJ77k/AF69c3i3JH6DmmTxH1Kff4w=","iv":"8m8VfEeiyyBf6PTHdcKFVw==","data":"7B/LVElTJlN5YYnYyYaeht9WKr/vZftXx/O65HI/Uqn+02kxkYHKgLdRWJiyo0SQFapqgWKltuOW6Qce/vxOdJeG6yCqqI/UUF254o+YgAeDuI2nh7egDTJZhxN3RuTGJXOwIX54ykWK30AajAJWcTaC6cgGmc418EUauLu2mLXQf0i0s5cou/lZDtmjn0ZqQ35LNvB6BoMcfwSg4TVtW87kqELnAtbErfF22Cx9L1xfyyR7xcbMzrHXejBNGnKehnXGHd1YnH7DC+oOppjYloJ577+d/bcYZiYWmZOWhlOqcgg2rHg8rGgDKuPhyd5LVaydQ+HTOI+1Z7AetZ9szO/xfdUm46HCq6NjjTARD6CjLIUb+9aX1o9XUi7o6Fd1o6d8ATlEO1wRH6KdXK9QBhbyEgw4HL1IryChWK3ETvq9OS77yEo0sNpHLFijK/W+hV93oqaB7DniIFn21IzgxQ=="})
// @Success 200 {object} string
// @Failure 500 {object} string
// @Router /v1/migration/sync/sync [post]
func sync(wr http.ResponseWriter, r *http.Request) {
	//wr.Header().Add("Access-Control-Allow-Origin", "*")
	//wr.Header().Add("Access-Control-Allow-Credentials", "true")
	//wr.Header().Add("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
	//wr.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")

	var syncConfig model.SyncConfig
	var err error
	if config.Env.IsEncryption == "true" {
		var payload model.HybridPayload
		_ = json.NewDecoder(r.Body).Decode(&payload)
		syncConfig, err = decodeSyncConfig(payload)

		if err != nil {
			wr.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(wr, err)
			return
		}
	} else {
		err = json.NewDecoder(r.Body).Decode(&syncConfig)
		if err != nil {
			wr.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(wr, err)
			return
		}
	}

	//var syncConfig model.SyncConfig
	//err := json.NewDecoder(r.Body).Decode(&syncConfig)

	//      sync 내용 시작
	rcloneInitialize()

	srcfs := ":" + syncConfig.Src.StorageType + ",access_key_id=" + syncConfig.Src.AccessKeyId + ",secret_access_key=" + syncConfig.Src.SecretAccessKey + ",endpoint=\"" + syncConfig.Src.Endpoint + "\":"
	if syncConfig.Src.Bucket != "" {
		srcfs = srcfs + syncConfig.Src.Bucket
	}
	dstfs := ":" + syncConfig.Dst.StorageType + ",access_key_id=" + syncConfig.Dst.AccessKeyId + ",secret_access_key=" + syncConfig.Dst.SecretAccessKey + ",endpoint=\"" + syncConfig.Dst.Endpoint + "\":"
	if syncConfig.Dst.Bucket != "" {
		dstfs = dstfs + syncConfig.Dst.Bucket
	}

	var syncRequest = model.SyncRequest{
		SrcFs: srcfs,
		DstFs: dstfs,
	}

	syncRequestJSON, err := json.Marshal(syncRequest)
	if err != nil {
		wr.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(wr, err)
		return
	}

	out, status := rcloneRPC("sync/sync", string(syncRequestJSON))
	// sync 내용 시작 끝

	wr.Header().Add("Content-type", "application/json")
	var resultjson map[string]interface{}
	json.Unmarshal([]byte(out), &resultjson)

	if status == 200 {
		wr.WriteHeader(status)
		fmt.Fprint(wr, resultjson)
	} else {
		wr.WriteHeader(status)
		fmt.Println(resultjson["error"].(string))
		fmt.Fprint(wr, errors.New("an unknown error occurred"))
	}
}

// @Summary Copying Between Storage
// @Description Copying Between Storage.
// @Description Example request body before encoding :
// @Description {
// @Description     "src": {
// @Description         "storageType": "s3",
// @Description         "endpoint": "http://url.co.kr",
// @Description         "accessKeyId": "admin",
// @Description         "secretAccessKey": "admin",
// @Description         "bucket": "abc"
// @Description     },
// @Description     "dst": {
// @Description         "storageType": "s3",
// @Description         "endpoint": "http://url.com",
// @Description         "accessKeyId": "admin",
// @Description         "secretAccessKey": "admin",
// @Description         "bucket": "abcd"
// @Description     }
// @Description }
// @Tags Migration
// @Accept json
// @Produce json
// @Param payload body model.HybridPayload true "encode base64 model.SyncConfig" SchemaExample({"key":"iCJcdWA198EF1uuzePq8M+Onsx8/4KJ9Mm9YDY+j+KUSxV+/4YyRkeDiuVAyRHkU8UUTyXS/zRhb2t8gUMKJiJaReLFgVyrweVMxRLk2Jx1yshNQ++Y/Cq4+Jvvk9lqsqpbmcoP2VrQoP4C4vymWQ+jwaV/vkJfUK0j6l6LlZenL8PmSzbhzO8kFVaS69TULu4IKbAK9YBeFN/QFhR84/St2fkd5RO8drgN0nPjieVlePc1N9mQewxkPQFXEsDAS3ws3xiBrw7YpHMn7mAk2VZ5y4F3vuEbGL7dIgWbG5UqGXdFGpyUnD91b7Tq+oMCrYhFIquOOIi3eqXirgmdZqOpmdBjDpM97t3oLKpCJDKTU9ceQSAVRSfP4HAN2LSbHTj/n4IZsVzpfcgTuXq0jHIRfrM3VPWnkidpI0v5cWMpJwAGvWcYmR7mwKCeSTJJ5/iE+0usbc/BaUQxFberLf2+QgIsNaR9V9iVnswwMQ0KVGGitsrEuCiTPv5Nlhi/WK8qvw42sFt098EPToWJ7/yHYAEEwt5Wr+4mAh3rNiLFfFB85JWM4ScrHyufHcHU4weqb/Bg5MD2ctO5fL7sp0vSOLSS3ziBpganU8Wzj7rLbaMrJbrhyZuhm6d7XdH/TvROFbTKO5bXHxwOJ77k/AF69c3i3JH6DmmTxH1Kff4w=","iv":"8m8VfEeiyyBf6PTHdcKFVw==","data":"7B/LVElTJlN5YYnYyYaeht9WKr/vZftXx/O65HI/Uqn+02kxkYHKgLdRWJiyo0SQFapqgWKltuOW6Qce/vxOdJeG6yCqqI/UUF254o+YgAeDuI2nh7egDTJZhxN3RuTGJXOwIX54ykWK30AajAJWcTaC6cgGmc418EUauLu2mLXQf0i0s5cou/lZDtmjn0ZqQ35LNvB6BoMcfwSg4TVtW87kqELnAtbErfF22Cx9L1xfyyR7xcbMzrHXejBNGnKehnXGHd1YnH7DC+oOppjYloJ577+d/bcYZiYWmZOWhlOqcgg2rHg8rGgDKuPhyd5LVaydQ+HTOI+1Z7AetZ9szO/xfdUm46HCq6NjjTARD6CjLIUb+9aX1o9XUi7o6Fd1o6d8ATlEO1wRH6KdXK9QBhbyEgw4HL1IryChWK3ETvq9OS77yEo0sNpHLFijK/W+hV93oqaB7DniIFn21IzgxQ=="})
// @Success 200 {object} string
// @Failure 500 {object} string
// @Router /v1/migration/sync/copy [post]
func copy(wr http.ResponseWriter, r *http.Request) {
	var syncConfig model.SyncConfig
	var err error
	if config.Env.IsEncryption == "true" {
		var payload model.HybridPayload
		_ = json.NewDecoder(r.Body).Decode(&payload)
		syncConfig, err = decodeSyncConfig(payload)

		if err != nil {
			wr.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(wr, err)
			return
		}
	} else {
		err = json.NewDecoder(r.Body).Decode(&syncConfig)
		if err != nil {
			wr.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(wr, err)
			return
		}
	}

	rcloneInitialize()
	srcfs := ":" + syncConfig.Src.StorageType + ",access_key_id=" + syncConfig.Src.AccessKeyId + ",secret_access_key=" + syncConfig.Src.SecretAccessKey + ",endpoint=\"" + syncConfig.Src.Endpoint + "\":"
	if syncConfig.Src.Bucket != "" {
		srcfs = srcfs + syncConfig.Src.Bucket
	}
	dstfs := ":" + syncConfig.Dst.StorageType + ",access_key_id=" + syncConfig.Dst.AccessKeyId + ",secret_access_key=" + syncConfig.Dst.SecretAccessKey + ",endpoint=\"" + syncConfig.Dst.Endpoint + "\":"
	if syncConfig.Dst.Bucket != "" {
		dstfs = dstfs + syncConfig.Dst.Bucket
	}
	var syncRequest = model.SyncRequest{
		SrcFs: srcfs,
		DstFs: dstfs,
	}

	syncRequestJSON, err := json.Marshal(syncRequest)
	if err != nil {
		wr.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(wr, err)
		return
	}

	out, status := rcloneRPC("sync/copy", string(syncRequestJSON))

	wr.Header().Add("Content-type", "application/json")

	var resultjson map[string]interface{}
	json.Unmarshal([]byte(out), &resultjson)

	if status == 200 {
		wr.WriteHeader(status)
		fmt.Fprint(wr, resultjson)
	} else {
		wr.WriteHeader(status)
		fmt.Println(resultjson["error"].(string))
		fmt.Fprint(wr, errors.New("an unknown error occurred"))
	}
}

// @Summary Check bucket list
// @Description Check bucket list.
// @Description Example request body before encoding :
// @Description 	{
// @Description 		"storageType": "s3",
// @Description 		"endpoint": "http://url.com",
// @Description 		"accessKeyId": "admin",
// @Description 		"secretAccessKey": "tpsxj0812",
// @Description 		"bucket": ""
// @Description 	}
// @Tags Migration
// @Accept json
// @Produce json
// @Param payload body model.HybridPayload true "encode base64 model.StorageConfig" SchemaExample({"key":"iCJcdWA198EF1uuzePq8M+Onsx8/4KJ9Mm9YDY+j+KUSxV+/4YyRkeDiuVAyRHkU8UUTyXS/zRhb2t8gUMKJiJaReLFgVyrweVMxRLk2Jx1yshNQ++Y/Cq4+Jvvk9lqsqpbmcoP2VrQoP4C4vymWQ+jwaV/vkJfUK0j6l6LlZenL8PmSzbhzO8kFVaS69TULu4IKbAK9YBeFN/QFhR84/St2fkd5RO8drgN0nPjieVlePc1N9mQewxkPQFXEsDAS3ws3xiBrw7YpHMn7mAk2VZ5y4F3vuEbGL7dIgWbG5UqGXdFGpyUnD91b7Tq+oMCrYhFIquOOIi3eqXirgmdZqOpmdBjDpM97t3oLKpCJDKTU9ceQSAVRSfP4HAN2LSbHTj/n4IZsVzpfcgTuXq0jHIRfrM3VPWnkidpI0v5cWMpJwAGvWcYmR7mwKCeSTJJ5/iE+0usbc/BaUQxFberLf2+QgIsNaR9V9iVnswwMQ0KVGGitsrEuCiTPv5Nlhi/WK8qvw42sFt098EPToWJ7/yHYAEEwt5Wr+4mAh3rNiLFfFB85JWM4ScrHyufHcHU4weqb/Bg5MD2ctO5fL7sp0vSOLSS3ziBpganU8Wzj7rLbaMrJbrhyZuhm6d7XdH/TvROFbTKO5bXHxwOJ77k/AF69c3i3JH6DmmTxH1Kff4w=","iv":"8m8VfEeiyyBf6PTHdcKFVw==","data":"7B/LVElTJlN5YYnYyYaeht9WKr/vZftXx/O65HI/Uqn+02kxkYHKgLdRWJiyo0SQFapqgWKltuOW6Qce/vxOdJeG6yCqqI/UUF254o+YgAeDuI2nh7egDTJZhxN3RuTGJXOwIX54ykWK30AajAJWcTaC6cgGmc418EUauLu2mLXQf0i0s5cou/lZDtmjn0ZqQ35LNvB6BoMcfwSg4TVtW87kqELnAtbErfF22Cx9L1xfyyR7xcbMzrHXejBNGnKehnXGHd1YnH7DC+oOppjYloJ577+d/bcYZiYWmZOWhlOqcgg2rHg8rGgDKuPhyd5LVaydQ+HTOI+1Z7AetZ9szO/xfdUm46HCq6NjjTARD6CjLIUb+9aX1o9XUi7o6Fd1o6d8ATlEO1wRH6KdXK9QBhbyEgw4HL1IryChWK3ETvq9OS77yEo0sNpHLFijK/W+hV93oqaB7DniIFn21IzgxQ=="})
// @Success 200 {array} object
// @Failure 404 {object} string
// @Failure 500 {object} string
// @Router /v1/migration/operations/list [post]
func bucketList(wr http.ResponseWriter, r *http.Request) {
	var storageConfig model.StorageConfig
	var err error
	if config.Env.IsEncryption == "true" {
		var payload model.HybridPayload
		_ = json.NewDecoder(r.Body).Decode(&payload)
		storageConfig, err = decodeStorageConfig(payload)
		if err != nil {
			wr.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(wr, err)
			return
		}
	} else {
		err = json.NewDecoder(r.Body).Decode(&storageConfig)
		if err != nil {
			wr.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(wr, err)
			return
		}
	}

	rcloneInitialize()
	fs := ":" + storageConfig.StorageType + ",access_key_id=" + storageConfig.AccessKeyId + ",secret_access_key=" + storageConfig.SecretAccessKey + ",endpoint=\"" + storageConfig.Endpoint + "\":"

	var listRequest = model.ListRequest{
		Fs:     fs,
		Remote: "",
		Opts:   "dirsOnly: true",
	}

	requestJSON, err := json.Marshal(listRequest)
	if err != nil {
		wr.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(wr, err)
		return
	}

	out, status := rcloneRPC("operations/list", string(requestJSON))

	wr.Header().Add("Content-type", "application/json")

	var resultjson map[string]interface{}
	json.Unmarshal([]byte(out), &resultjson)

	if status == 200 {
		wr.WriteHeader(status)
		fmt.Fprint(wr, resultjson)
	} else {
		wr.WriteHeader(status)
		fmt.Println(resultjson["error"].(string))
		fmt.Fprint(wr, errors.New("an unknown error occurred"))
	}
}
