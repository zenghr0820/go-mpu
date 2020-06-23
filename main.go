package main

import (
	"bufio"
	"log"
	"net"
	"os"
	"time"

	"cloud-disk/container/flv"
	"cloud-disk/parser"
	"cloud-disk/parser/h264"
)

//var wg sync.WaitGroup

func main() {

	receiveRtp()

}

func receiveRtp() {
	ip := "0.0.0.0:5766"
	udpAddr, err := net.ResolveUDPAddr("udp", ip)
	if err != nil {
		panic(err)
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		panic(err)
	}

	f, err := os.Create("./h264toflv.flv") //创建文件
	if err != nil {
		panic(err)
	}
	w := bufio.NewWriter(f) //创建新的 Writer 对象

	// H264 解析器
	h264Parser := h264.NewParser()

	// 时间戳计算
	var cts uint32 = 0

	// 定时器
	t := time.NewTimer(time.Second * 10)

	// h264 to flv
	go func() {
		var err error
		//once := new(sync.Once)
		// 封装 flv，写入文件
		// 1. 写入 flv 头部
		_, err = w.Write(flv.NewFlvHeader().ToByte())
		if err != nil {
			panic(err)
		}

		// 2. 写入 flv payload （tag size + tag）
		// 3. 写入上一个 Tag 长度
		_, err = w.Write([]byte{0x00, 0x00, 0x00, 0x00})
		if err != nil {
			panic(err)
		}

		// 4. todo 写入 元数据 Tag
		metaTag := flv.NewTagPacket(flv.TAG_TYPE_SCRIPT)
		metaTag.TagType = 0x12
		meta := flv.NewMetaTagData()
		meta.Init()
		metaTag.SetPayload(meta)

		_, err = w.Write(metaTag.Bytes())
		if err != nil {
			panic(err)
		}

		//num := 0

		for {
			select {
			case pack, ok := <-h264Parser.NaluCache:
				if !ok {
					return
				}

				tagPack := flv.NewTagPacket(flv.TAG_TYPE_VIDEO)
				videoTagPack := flv.NewAvcVideoPacket()

				// 计算 tag data 大小
				if pack.IsSps() {
					// 获取下一个 pack
					ppsPack := <-h264Parser.NaluCache
					if !ppsPack.IsPps() {
						//log.Println(pack.Data)
						//log.Println(ppsPack.Data)
						panic("sps and pps error")
					}

					log.Println("======= 封装.视频同步包 =======")
					videoTagPack.AvcPacketType = 0x00
					// 5. AVCPacketType = 0
					// 封装 AVCDecoderConfigurationRecord
					sequenceHeader := flv.NewAVCDecoderConfigurationRecord()
					sequenceHeader.AvcProfileIndication = pack.Data[1]
					sequenceHeader.ProfileCompatility = pack.Data[2]
					sequenceHeader.AvcLevelIndication = pack.Data[3]
					// sps
					sequenceHeader.SpsNum = 0x01
					sequenceHeader.SpsSize = uint(pack.Length())
					sequenceHeader.Sps = pack.Data
					// pps
					sequenceHeader.PpsNum = 0x01
					sequenceHeader.PpsSize = uint(ppsPack.Length())
					sequenceHeader.Pps = ppsPack.Data

					// 追加
					videoTagPack.SetData(sequenceHeader)
					tagPack.Payload = videoTagPack
				} else if pack.IsSet() {
					// SET 数据要跟 下一个 nalu 封装在一个 Tag
					// 获取下一个 pack
					nextPack := <-h264Parser.NaluCache

					// 追加 VideoTag
					videoTagPack.FrameType = 0x01
					videoTagPack.AvcPacketType = 0x01

					unitData := flv.NewNaluUnitData()
					unitData.SetData(pack)
					unitData.SetData(nextPack)
					videoTagPack.SetData(unitData)
					tagPack.Payload = videoTagPack

				} else {
					// 追加 VideoTag
					if pack.IsKey() {
						videoTagPack.FrameType = 0x01
						videoTagPack.AvcPacketType = 0x01
						//videoTagPack.SetCompositionTime(40)
					} else {
						videoTagPack.FrameType = 0x02
						videoTagPack.AvcPacketType = 0x01
						//videoTagPack.SetCompositionTime(40)
					}

					unitData := flv.NewNaluUnitData()
					unitData.SetData(pack)
					videoTagPack.SetData(unitData)
					tagPack.Payload = videoTagPack
				}

				// 计算 Tag header
				// TagType 0x09
				tagPack.TagType = flv.TAG_TYPE_VIDEO

				// Timestamp 裸 h.264 没有时间戳，默认 25fps，即40ms一帧数据。
				tagPack.Ts = cts
				cts += 40

				_, err = w.Write(tagPack.Bytes())
				if err != nil {
					panic(err)
				}

			case <-t.C:
				log.Println("解析结束 END ==========================>")
				return
				//fmt.Println(tagSizeBuf)
			}

			w.Flush()
		}

	}()

	for {
		buff := make([]byte, 2*1024)
		num, err := conn.Read(buff)
		if err != nil {
			continue
		}

		data := buff[:num]
		rtpParser := parser.NewRtpParser()
		rtpPack := rtpParser.Parse(data)

		// 提取 h.264
		if rtpPack.PayloadType == 96 { // ps 荷载 h.264
			h264Real := rtpPack.RealData()
			if len(h264Real) <= 0 {
				continue
			}
			// 解析 h264
			h264Parser.WriteByte(h264Real)

			// 刷新定时器
			t.Reset(time.Second * 10)

			//n4, err1 := w.Write(h264)
			//if err1 != nil {
			//	panic(err1)
			//}
			//w.Flush()
			//fmt.Printf("写入 %d 个字节 \n", n4)
		}

		//log.Println(rtpPack)
	}
	f.Close()

}
