package model

type StorageConfig struct {
	AccessKeyId     string `json:"accessKeyId"`
	Bucket          string `json:"bucket"`
	Endpoint        string `json:"endpoint"`
	SecretAccessKey string `json:"secretAccessKey"`
	StorageType     string `json:"storageType"`
}

type SyncConfig struct {
	Dst StorageConfig `json:"dst"`
	Src StorageConfig `json:"src"`
}

type SyncRequest struct {
	DstFs string `json:"dstFs"`
	SrcFs string `json:"srcFs"`
}

type ListRequest struct {
	Fs     string `json:"fs"`
	Remote string `json:"remote"`
	Opts   string `json:"opts"`
}
