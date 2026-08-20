package config

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server ServerConfig
	MySQL  MySQLConfig
	Redis  RedisConfig
}

type ServerConfig struct {
	Port int
}

type MySQLConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

var AppConfig Config

func LoadConfig() {
	loadDotEnv()

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")

	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("读取配置文件失败: %v", err)
	}

	err = viper.Unmarshal(&AppConfig)
	if err != nil {
		log.Fatalf("解析配置失败: %v", err)
	}

	applyEnvOverrides()

	fmt.Println("配置加载成功")
}

// 读取环境变量
func loadDotEnv() {
	file, err := os.Open(".env")
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key != "" && os.Getenv(key) == "" {
			_ = os.Setenv(key, value)
		}
	}
}

// 环境变量覆盖
func applyEnvOverrides() {
	if value := os.Getenv("SERVER_PORT"); value != "" {
		if port, err := strconv.Atoi(value); err == nil {
			AppConfig.Server.Port = port
		}
	}
	if value := os.Getenv("MYSQL_HOST"); value != "" {
		AppConfig.MySQL.Host = value
	}
	if value := os.Getenv("MYSQL_PORT"); value != "" {
		if port, err := strconv.Atoi(value); err == nil {
			AppConfig.MySQL.Port = port
		}
	}
	if value := os.Getenv("MYSQL_USERNAME"); value != "" {
		AppConfig.MySQL.Username = value
	}
	if value := os.Getenv("MYSQL_PASSWORD"); value != "" {
		AppConfig.MySQL.Password = value
	}
	if value := os.Getenv("MYSQL_DATABASE"); value != "" {
		AppConfig.MySQL.Database = value
	}
	if value := os.Getenv("REDIS_HOST"); value != "" {
		AppConfig.Redis.Host = value
	}
	if value := os.Getenv("REDIS_PORT"); value != "" {
		if port, err := strconv.Atoi(value); err == nil {
			AppConfig.Redis.Port = port
		}
	}
	if value := os.Getenv("REDIS_PASSWORD"); value != "" {
		AppConfig.Redis.Password = value
	}
	if value := os.Getenv("REDIS_DB"); value != "" {
		if db, err := strconv.Atoi(value); err == nil {
			AppConfig.Redis.DB = db
		}
	}
}
