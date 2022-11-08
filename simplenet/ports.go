package simplenet

import "sync/atomic"

var ringpool *PortRingPool

type PortRingPool struct {
	startPort uint32
	endPort   uint32
	curPort   uint32 // 当前端口
}

// InitPortRingPool 初始化端口池
func InitPortRingPool(startPort, endPort int) {
	ringpool = NewPortRingPool(startPort, endPort)
}

// NewPortRingPool 创建一个新的 PortPool 实例
func NewPortRingPool(startPort, endPort int) *PortRingPool {
	if endPort <= startPort {
		panic("endPort must be greater than startPort")
	}
	return &PortRingPool{
		startPort: uint32(startPort),
		endPort:   uint32(endPort),
		curPort:   uint32(startPort),
	}
}

// Acquire 获取环形端口
func (p *PortRingPool) acquire() int32 {
	// 计算端口范围
	rangeSize := p.endPort - p.startPort + 1
	// 递增当前端口并计算下一个可用端口
	currentPort := atomic.AddUint32(&p.curPort, 1) - 1
	sourcePort := p.startPort + uint32(currentPort%uint32(rangeSize))
	return int32(sourcePort)
}

func Acquire() int {
	return int(ringpool.acquire())
}
