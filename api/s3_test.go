package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"kps-migration-api/config"
	"kps-migration-api/model"
)

func withRcloneMock(t *testing.T, fn func(initCalled *bool, lastMethod *string, lastPayload *string, setStatus func(int), setOut func(string))) {
	t.Helper()

	oldInit := rcloneInitialize
	oldRPC := rcloneRPC
	defer func() {
		rcloneInitialize = oldInit
		rcloneRPC = oldRPC
	}()

	initCalled := false
	lastMethod := ""
	lastPayload := ""
	status := 200
	out := `{"result":"ok"}`

	rcloneInitialize = func() {
		initCalled = true
	}

	rcloneRPC = func(method, in string) (string, int) {
		lastMethod = method
		lastPayload = in
		return out, status
	}

	fn(&initCalled, &lastMethod, &lastPayload,
		func(s int) { status = s },
		func(o string) { out = o },
	)
}

func TestSync_PlainJSON_Success(t *testing.T) {
	if config.Env != nil {
		config.Env.IsEncryption = "false"
	}

	withRcloneMock(t, func(initCalled *bool, lastMethod *string, lastPayload *string, setStatus func(int), setOut func(string)) {
		syncCfg := model.SyncConfig{
			Src: model.StorageConfig{
				StorageType:     "s3",
				Endpoint:        "http://src-endpoint",
				AccessKeyId:     "srcKey",
				SecretAccessKey: "srcSecret",
				Bucket:          "src-bucket",
			},
			Dst: model.StorageConfig{
				StorageType:     "s3",
				Endpoint:        "http://dst-endpoint",
				AccessKeyId:     "dstKey",
				SecretAccessKey: "dstSecret",
				Bucket:          "dst-bucket",
			},
		}
		body, _ := json.Marshal(syncCfg)

		req := httptest.NewRequest(http.MethodPost, "/v1/migration/sync/sync", bytes.NewReader(body))
		w := httptest.NewRecorder()

		sync(w, req)

		res := w.Result()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", res.StatusCode)
		}

		if !*initCalled {
			t.Fatalf("expected rcloneInitialize to be called")
		}
		if *lastMethod != "sync/sync" {
			t.Fatalf("expected method sync/sync, got %q", *lastMethod)
		}
		if *lastPayload == "" {
			t.Fatalf("expected RPC payload to be non-empty")
		}
	})
}

func TestSync_InvalidJSON_Returns500(t *testing.T) {
	if config.Env != nil {
		config.Env.IsEncryption = "false"
	}

	body := []byte("not-json")
	req := httptest.NewRequest(http.MethodPost, "/v1/migration/sync/sync", bytes.NewReader(body))
	w := httptest.NewRecorder()

	sync(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", res.StatusCode)
	}
}

func TestCopy_PlainJSON_UsesSyncCopyRPC(t *testing.T) {
	if config.Env != nil {
		config.Env.IsEncryption = "false"
	}

	withRcloneMock(t, func(initCalled *bool, lastMethod *string, lastPayload *string, setStatus func(int), setOut func(string)) {
		syncCfg := model.SyncConfig{
			Src: model.StorageConfig{
				StorageType:     "s3",
				Endpoint:        "http://src-endpoint",
				AccessKeyId:     "srcKey",
				SecretAccessKey: "srcSecret",
			},
			Dst: model.StorageConfig{
				StorageType:     "s3",
				Endpoint:        "http://dst-endpoint",
				AccessKeyId:     "dstKey",
				SecretAccessKey: "dstSecret",
			},
		}
		body, _ := json.Marshal(syncCfg)

		req := httptest.NewRequest(http.MethodPost, "/v1/migration/sync/copy", bytes.NewReader(body))
		w := httptest.NewRecorder()

		copy(w, req)

		res := w.Result()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", res.StatusCode)
		}
		if !*initCalled {
			t.Fatalf("expected rcloneInitialize to be called")
		}
		if *lastMethod != "sync/copy" {
			t.Fatalf("expected method sync/copy, got %q", *lastMethod)
		}
		if *lastPayload == "" {
			t.Fatalf("expected RPC payload to be non-empty")
		}
	})
}

func TestBucketList_PlainJSON_Success(t *testing.T) {
	if config.Env != nil {
		config.Env.IsEncryption = "false"
	}

	withRcloneMock(t, func(initCalled *bool, lastMethod *string, lastPayload *string, setStatus func(int), setOut func(string)) {
		storageCfg := model.StorageConfig{
			StorageType:     "s3",
			Endpoint:        "http://endpoint",
			AccessKeyId:     "key",
			SecretAccessKey: "secret",
			Bucket:          "",
		}
		body, _ := json.Marshal(storageCfg)

		setOut(`{"buckets":[{"name":"bucket1"}]}`)
		setStatus(200)

		req := httptest.NewRequest(http.MethodPost, "/v1/migration/operations/list", bytes.NewReader(body))
		w := httptest.NewRecorder()

		bucketList(w, req)

		res := w.Result()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", res.StatusCode)
		}
		if !*initCalled {
			t.Fatalf("expected rcloneInitialize to be called")
		}
		if *lastMethod != "operations/list" {
			t.Fatalf("expected method operations/list, got %q", *lastMethod)
		}
	})
}

func TestBucketList_RcloneError_Returns500(t *testing.T) {
	if config.Env != nil {
		config.Env.IsEncryption = "false"
	}

	withRcloneMock(t, func(initCalled *bool, lastMethod *string, lastPayload *string, setStatus func(int), setOut func(string)) {
		storageCfg := model.StorageConfig{
			StorageType:     "s3",
			Endpoint:        "http://endpoint",
			AccessKeyId:     "key",
			SecretAccessKey: "secret",
			Bucket:          "",
		}
		body, _ := json.Marshal(storageCfg)

		setOut(`{"error":"something bad"}`)
		setStatus(500)

		req := httptest.NewRequest(http.MethodPost, "/v1/migration/operations/list", bytes.NewReader(body))
		w := httptest.NewRecorder()

		bucketList(w, req)

		res := w.Result()
		if res.StatusCode != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", res.StatusCode)
		}
	})
}

func TestSync_Encrypted_DecodeError_Returns500(t *testing.T) {
	if config.Env != nil {
		config.Env.IsEncryption = "true"
	}

	oldDecode := decodeSyncConfig
	defer func() { decodeSyncConfig = oldDecode }()

	decodeSyncConfig = func(payload model.HybridPayload) (model.SyncConfig, error) {
		return model.SyncConfig{}, errors.New("decode failed")
	}

	body := []byte(`{"encryptedData":"x"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/migration/sync/sync", bytes.NewReader(body))
	w := httptest.NewRecorder()

	sync(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", res.StatusCode)
	}
}

func TestCopy_InvalidJSON_Returns500(t *testing.T) {
	if config.Env != nil {
		config.Env.IsEncryption = "false"
	}

	body := []byte("not-json")
	req := httptest.NewRequest(http.MethodPost, "/v1/migration/sync/copy", bytes.NewReader(body))
	w := httptest.NewRecorder()

	copy(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", res.StatusCode)
	}
}

func TestBucketList_Encrypted_UsesDecodeStorageConfigAndRPC(t *testing.T) {
	if config.Env != nil {
		config.Env.IsEncryption = "true"
	}

	oldDecode := decodeStorageConfig
	defer func() { decodeStorageConfig = oldDecode }()

	decodeStorageConfig = func(payload model.HybridPayload) (model.StorageConfig, error) {
		return model.StorageConfig{
			StorageType:     "s3",
			Endpoint:        "http://endpoint",
			AccessKeyId:     "key",
			SecretAccessKey: "secret",
		}, nil
	}

	withRcloneMock(t, func(initCalled *bool, lastMethod *string, lastPayload *string, setStatus func(int), setOut func(string)) {
		setStatus(200)
		setOut(`{"buckets":[{"name":"bucket1"}]}`)

		body := []byte(`{"encryptedData":"x"}`)
		req := httptest.NewRequest(http.MethodPost, "/v1/migration/operations/list", bytes.NewReader(body))
		w := httptest.NewRecorder()

		bucketList(w, req)

		res := w.Result()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", res.StatusCode)
		}
		if !*initCalled {
			t.Fatalf("expected rcloneInitialize to be called")
		}
		if *lastMethod != "operations/list" {
			t.Fatalf("expected method operations/list, got %q", *lastMethod)
		}
	})
}
