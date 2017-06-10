package model

type ConnectData struct {
	Filename string `json:"filename"`
	FilePath string `json:"filepath"`
	FileMD5  string `json:"filemd5"`
	Status   int    `json:"status"`
}

type ReturnData struct {
	Filename    string `json:"filename"`
	Status      int    `json:"status"`
	PacketIndex int    `json:"packet_index"`
}

type FileData struct {
	FilePackets int
	PacketIndex int
	FileOffset  int
	Filename    string
	FilePath    string
	PacketMD5   string
	Data        string
}
