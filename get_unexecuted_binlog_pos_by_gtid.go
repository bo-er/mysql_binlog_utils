package mysql_binlog_utils

import (
	"encoding/binary"
	gtid "github.com/ikarishinjieva/go-gtid"
	"io"
	"os"
	"strconv"
)

func GetUnexecutedBinlogPosByGtid(binlogPath string, executedGtidDesc string, includeEventBeforeFirst bool) (pos uint, err error) {
	file, err := os.Open(binlogPath)
	if nil != err {
		return 0, err
	}
	defer file.Close()

	p := uint32(4)
	headerBs := make([]byte, 19)
	payloadBs := make([]byte, 1024)
	lastExecutedGtidPos := uint32(0)

	for {
		if _, err := file.Seek(int64(p), 0); nil != err {
			return 0, err
		}

		if _, err := io.ReadFull(file, headerBs); nil != err {
			return 0, err
		}

		length := binary.LittleEndian.Uint32(headerBs[9:13])
		eventType := int(headerBs[4])

		if GTID_LOG_EVENT != eventType {
			p += length
			continue
		}

		payloadLength := length - 19
		if payloadLength > uint32(len(payloadBs)) {
			payloadBs = make([]byte, payloadLength)
		}

		if _, err := io.ReadFull(file, payloadBs[:payloadLength]); nil != err {
			return 0, err
		}

		uuid := bytesToUuid(payloadBs[1:17])
		number := bytesToUint64(payloadBs[17:25])
		g := uuid + ":" + strconv.FormatUint(number, 10)
		contain, err := gtid.GtidContain(executedGtidDesc, g)
		if nil != err {
			return 0, err
		}
		if contain {
			lastExecutedGtidPos = p
			p += length
		} else {
			retPos := p
			if includeEventBeforeFirst && 0 != lastExecutedGtidPos {
				retPos = lastExecutedGtidPos
			}
			return uint(retPos), nil
		}
	}
}
