package config

import (
	"fmt"
	"time"

	"github.com/importcjj/ddxq/internal/boost"
	"github.com/importcjj/ddxq/pkg/api"
	"github.com/importcjj/ddxq/pkg/dingding"
	"github.com/jinzhu/configor"
)

type Config struct {
	API             api.Config `yaml:"api" json:"api"`
	CartInterval    string     `yaml:"cart_interval" json:"cart_interval" default:"2m"`
	ReserveInterval string     `yaml:"reserve_interval" json:"reserve_interval" default:"2s"`

	Dingding  dingding.Config `yaml:"dingding" json:"dingding"`
	BoostMode boost.Config    `yaml:"boost_mode" json:"boost_mode"`
}

func Load(filepath string) (Config, error) {
	var config Config
	err := configor.Load(&config, filepath)

	if err != nil {
		return config, fmt.Errorf("解析配置文件失败: %w", err)
	}

	return config, nil
}

func (c *Config) NewMode() (*Mode, error) {
	boostMode, err := boost.New(c.BoostMode)
	if err != nil {
		return nil, fmt.Errorf("无法创建boost: %w", err)
	}

	cartInterval, err := time.ParseDuration(c.CartInterval)
	if err != nil {
		return nil, err
	}

	reserveInterval, err := time.ParseDuration(c.ReserveInterval)
	if err != nil {
		return nil, err
	}

	mode := &Mode{
		BoostMode:       *boostMode,
		cartInterval:    cartInterval,
		reserveInterval: reserveInterval,
	}

	return mode, nil
}