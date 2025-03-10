// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/config"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/container"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/pkg/logging"
	"github.com/mercedes-benz/car-integrated-service-mesh-architecture/web"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 1024
	statusPeriod   = 30 * time.Second
)

type statusManager struct {
	containerManager container.Manager
}

type message struct {
	Action string
	ID     string
}

func (s statusManager) sendStatus(ws *websocket.Conn) {
	ticker := time.NewTicker(statusPeriod)
	defer ticker.Stop()

	sendContainerList := func() error {
		containers, err := s.containerManager.Containers(context.Background())
		logging.LogErr(err)

		if err != nil {
			return nil
		}

		message, err := json.MarshalIndent(containers, "", "    ")
		logging.LogErr(err)

		if err != nil {
			return nil
		}

		if err := ws.WriteMessage(1, message); err != nil {
			if errors.Is(err, websocket.ErrCloseSent) {
				return nil
			}

			logging.LogErr(err)

			return err
		}

		return nil
	}

	err := sendContainerList()
	if err != nil {
		return
	}

	for {
		select {
		case <-ticker.C:
			err = sendContainerList()
			if err != nil {
				return
			}
		}
	}
}

func (s statusManager) sendPing(ws *websocket.Conn) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait)); err != nil {
				if !errors.Is(err, websocket.ErrCloseSent) {
					logging.LogErr(err)
				}

				return
			}
		}
	}
}

func (s statusManager) readMessages(ws *websocket.Conn) {
	defer func() {
		_ = ws.Close()
	}()

	ws.SetReadLimit(maxMessageSize)
	_ = ws.SetReadDeadline(time.Now().Add(pongWait))

	ws.SetPongHandler(func(string) error {
		_ = ws.SetReadDeadline(time.Now().Add(pongWait))

		return nil
	})

	for {
		_, msgContent, err := ws.ReadMessage()
		if err != nil {
			// log everything but gone connections
			if !websocket.IsCloseError(err, websocket.CloseGoingAway) {
				logging.LogErr(err)
			}

			break
		}

		var msg message
		err = json.Unmarshal(msgContent, &msg)
		logging.LogErr(err)

		if err != nil {
			continue
		}

		switch strings.ToLower(msg.Action) {
		case "start":
			err = s.containerManager.StartContainer(context.Background(), msg.ID)
			logging.LogErr(err)

			if err != nil {
				continue
			}

		case "stop":
			err = s.containerManager.StopContainer(context.Background(), msg.ID)
			logging.LogErr(err)

			if err != nil {
				continue
			}
		}
	}
}

func (s statusManager) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	go s.sendStatus(ws)
	go s.readMessages(ws)
	go s.sendPing(ws)
}

func (s statusManager) handleIndex(w http.ResponseWriter, _ *http.Request) {
	t, err := template.ParseFS(web.Templates, "template/status-manager/index.tmpl")
	logging.LogErr(err)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	hostname, err := os.Hostname()
	logging.LogErr(err)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	err = t.Execute(w, hostname)
	logging.LogErr(err)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func Run() {
	cfg, err := config.New()
	logging.LogErr(err)

	if err != nil {
		return
	}

	if cfg.EnableDebugMode {
		logging.EnableDebugLogs()
	}

	var containerManager container.Manager
	if cfg.EmulateContainerRuntime {
		containerManager = container.NewDebugContainerManager(logging.DefaultDebugLevelWriter)
	} else {
		containerManager, err = container.NewDockerContainerManager(context.Background())
		logging.LogErr(err)

		if err != nil {
			return
		}
	}
	defer func() {
		logging.LogErr(containerManager.Close())
	}()

	sm := &statusManager{containerManager: containerManager}

	r := mux.NewRouter()
	r.HandleFunc("/", sm.handleIndex)
	r.HandleFunc("/ws", sm.handleWebSocket)

	assetDir, err := fs.Sub(web.Assets, "static")
	if err != nil {
		logging.DefaultLogger.Error().Err(err).Msg("")

		return
	}

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.FS(assetDir))))
	server := &http.Server{
		Addr:    ":" + strconv.Itoa(cfg.StatusMgrPort),
		Handler: r,
	}

	logging.DefaultLogger.Info().Msgf("Starting CARISMA status manager at port %d 🚀", cfg.StatusMgrPort)
	if err := server.ListenAndServe(); err != nil {
		logging.DefaultLogger.Error().Err(err).Msg("")
	}
}
