package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	uuid "github.com/satori/go.uuid"
	"github.com/shirou/gopsutil/cpu"
)

var serverID string

func init() {
	uuid := uuid.NewV4()
	serverID = fmt.Sprintf("%s", uuid)
}

type param string

const (
	// ServerID parameter for serverid comment
	ServerID param = "serverid"
	// CPUUsage parameter for cpuusage
	CPUUsage param = "cpu"
)

// Server is the server struct
type Server struct {
	DB     *sql.DB
	Logger zerolog.Logger
	mux    *http.ServeMux
}

func (server *Server) migrateTables() {
	lg := server.Logger.With().Str("context", "migrating table schema").Logger()
	lg.Info().Msg("Initializing Tables")
}

func (server *Server) monitorCPUUsage() {
	lg := server.Logger.With().Str("context", "monitor usage").Logger()
	for {
		percent, err := cpu.Percent(time.Second, false)
		if err != nil {
			log.Fatal(err)
		}
		if percent[0] > 70 {
			lg.Info().Msg("cpu usage greater than 70 percent")
		}
		if getEnv("SENDUSAGE", "0") == "1" {
			rawurl := getEnv("HITURL", "")
			URL, err := url.Parse(rawurl)
			if err != nil {
				lg.Err(err).Msg("Unable to parse url")
				return
			}
			q := URL.Query()
			for v := range q {
				switch v {
				case string(ServerID):
					q.Set("serverid", serverID)
				case string(CPUUsage):
					q.Set("cpu", fmt.Sprintf("%v", percent[0]))
				}
			}
			URL.RawQuery = q.Encode()
			http.Get(URL.String())
		}
		time.Sleep(5 * time.Second)
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func (server *Server) makeDBConnection() {
	DbUser := getEnv("DB_USER", "riddhi")
	DbPassword := getEnv("DB_PASSWORD", "riddhi1132")
	DbHost := getEnv("MYSQL_URL", "127.0.0.1")
	DbPort := getEnv("DB_PORT", "3306")
	DbName := getEnv("DB_NAME", "itautomation")
	DBURL := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local", DbUser, DbPassword, DbHost, DbPort, DbName)

	var err error
	server.DB, err = sql.Open(getEnv("DB_DRIVER", "mysql"), DBURL)
	if err != nil {
		log.Fatal("error opening connection with mysql", err)
	}
}

func (server *Server) registerCPUUsage(w http.ResponseWriter, r *http.Request) {
	lg := server.Logger.With().Str("context", "registering CPU Usage").Logger()
	cpu := r.FormValue("cpu")
	serverID := r.FormValue("serverid")
	cpuUsage, err := strconv.ParseFloat(cpu, 64)
	if err != nil {
		lg.Err(err).Msg("unable to parse cpu usage string into float")
		return
	}
	if cpuUsage > 70 {
		lg.Info().Msg(fmt.Sprintf("server %s cpu usage is more than 70", serverID))
	}
	lg.Info().Msg(fmt.Sprintf("server %s cpu usage is %s", serverID, cpu))
}

func (server *Server) runServer() {
	server.mux = http.NewServeMux()
	server.mux.HandleFunc("/cpu/usage", server.registerCPUUsage)
	lg := server.Logger.With().Str("Context", "HTTP Server").Logger()
	lg.Info().Msg(fmt.Sprintf("Running server on port %s", getEnv("PORT", "8080")))
	if err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%s", getEnv("PORT", "8080")), server.mux); err != nil {
		log.Fatal("Unable to start server", err)
	}
}

func runServer() {
	// load environment variables from .env file in the same directory as the binary
	server := Server{Logger: zerolog.New(os.Stdout)}
	lg := server.Logger.With().Str("context", "running server").Logger()
	lg.Info().Str("run", "server").Msg("runserver Method")
	go func() { server.monitorCPUUsage() }()
	if err := godotenv.Load(); err != nil {
		lg.Info().
			Str("environment variables", fmt.Sprintf("%v", err)).
			Msg("error loading environment variables from .env file")
	}
	server.makeDBConnection()
	server.runServer()
}

func main() {
	runServer()
}
