// internal/state/helpers_v1.go
package state

import "bytes"

func trySplitConstVarConst(
	p []byte,
) (constA []byte, lane []byte, constB []byte, ok bool) {
	if len(p) < 3 || p[len(p)-1] != '\n' {
		return nil, nil, nil, false
	}

	idx := bytes.LastIndexByte(p, '=')
	if idx <= 0 || idx >= len(p)-2 {
		return nil, nil, nil, false
	}

	constA = append([]byte(nil), p[:idx+1]...)
	lane = append([]byte(nil), p[idx+1:len(p)-1]...)
	constB = []byte{'\n'}
	return constA, lane, constB, true
}

func tryExtractLane(
	p []byte,
	constA []byte,
	constB []byte,
) (lane []byte, ok bool) {
	if !bytes.HasPrefix(p, constA) || !bytes.HasSuffix(p, constB) {
		return nil, false
	}

	start := len(constA)
	end := len(p) - len(constB)
	if end < start {
		return nil, false
	}

	lane = append([]byte(nil), p[start:end]...)
	return lane, true
}
