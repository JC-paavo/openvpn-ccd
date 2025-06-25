package utils

import (
	"encoding/binary"
	"errors"
	"net"
)

// 将IP地址转换为uint32整数
func ipToUint32(ip net.IP) (uint32, error) {
	if len(ip) == 16 { // 处理IPv6转IPv4映射地址
		ip = ip.To4()
	}
	if ip == nil {
		return 0, errors.New("无效的IP地址")
	}
	return binary.BigEndian.Uint32(ip), nil
}
func netMaskToUint32(mask net.IPMask) (uint32, error) {
	if mask == nil {
		return 0, errors.New("无效的IP地址")
	}
	return binary.BigEndian.Uint32(mask), nil
}

// 根据IP和子网掩码计算网段范围
func getNetworkRange(ip net.IP, mask net.IPMask) (uint32, uint32, error) {
	ipNum, err := ipToUint32(ip)
	if err != nil {
		return 0, 0, err
	}
	maskNum, err := netMaskToUint32(mask)
	if err != nil {
		return 0, 0, err
	}

	// 计算网络地址（IP & 掩码）
	network := ipNum & maskNum
	// 计算广播地址（网络地址 | 反掩码）
	broadcast := network | ^maskNum

	return network, broadcast, nil
}

// 判断两个网段是否冲突（IP+掩码格式）
func IsNetworkConflict(ip1, mask1, ip2, mask2 string) (bool, error) {

	//fmt.Println(ip1, mask1, ip2, mask2)
	// 解析IP和掩码
	ipObj1 := net.ParseIP(ip1)
	ipObj2 := net.ParseIP(ip2)
	maskObj1 := net.IPMask(net.ParseIP(mask1).To4())
	maskObj2 := net.IPMask(net.ParseIP(mask2).To4())

	if ipObj1 == nil || ipObj2 == nil || maskObj1 == nil || maskObj2 == nil {
		return false, errors.New("解析IP或掩码失败")
	}

	// 获取网段范围
	net1Start, net1End, err := getNetworkRange(ipObj1, maskObj1)
	if err != nil {
		return false, err
	}
	net2Start, net2End, err := getNetworkRange(ipObj2, maskObj2)
	if err != nil {
		return false, err
	}
	// 判断范围重叠：[a1,a2]和[b1,b2]重叠条件为 a1 <= b2 且 b1 <= a2
	return net1Start <= net2End && net2Start <= net1End, nil
}
