package model

type ConnectData struct {
	Filename    string `json:"filename"`
	FilePath    string `json:"filepath"`
	FileMD5     string `json:"filemd5"`
	FilePackets int    `json:"file_packets"`
	Status      int    `json:"status"`
}

type ReturnData struct {
	Filename    string `json:"filename"`
	Status      int    `json:"status"`
	PacketIndex int    `json:"packet_index"`
}

type FileData struct {
	PacketIndex int    `json:"packet_index"`
	FileOffset  int    `json:"file_offset"`
	Filename    string `json:"filename"`
	FilePath    string `json:"filepath"`
	Body        []byte `json:"body"`
	Status      int    `json:"status"`
}
