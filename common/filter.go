package common

import "github.com/liangyaopei/bloom"

var GlobalBloomFilter *bloom.Filter

func InitFilter() {
	GlobalBloomFilter = bloom.New(33554432, 10, true)
}
