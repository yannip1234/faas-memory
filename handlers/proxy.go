package handlers

import (
	"encoding/json"
	// b64 "encoding/base64"
	// "fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/btittelbach/go-bbhw"
	"bytes"
	
	// "strconv"

)

type response struct {
	Function     string
	ResponseBody string
	HostName     string
}

func gpio_turn_on(pin_num uint) error {
	pin, err := bbhw.NewSysfsGPIO(pin_num, bbhw.OUT)
	err = pin.SetState(true)
	time.Sleep(500 * time.Millisecond)
	err = pin.SetState(false)
	time.Sleep(500 * time.Millisecond)
	err = pin.SetState(true)
	return err
}
type Payload struct{
	Fid string `json:"fid"`
	Src string `json:"src"`
	Params string `json:"params,omitempty"`
	Lang string `json:"lang"`
	Worker string `json:"worker"`
}

type FuncCall struct{
	Params string `json:"params,omitempty"`
	Lang string `json:"lang,omitempty"`
	Worker string `json:"worker"`
}

// MakeProxy creates a proxy for HTTP web requests which can be routed to a function.
func MakeProxy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		name := vars["name"]
		log.Info("proxy request: " + name)

		v, okay := functions[name]
		if !okay {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("{ \"status\" : \"Not found\"}"))
			log.Errorf("%s not found", name)
			return
		}
		log.Info("V IMAGE IS!!!!!!! : " + v.Image)
		// Working GPIO pins
		worker_list := map[int]uint{
			1: 48, // works
			2: 67, // works
			3: 68, // works
		}

		gpio_turn_on(worker_list[3])

		v.InvocationCount = v.InvocationCount + 1

		defer r.Body.Close()
		body, _ := ioutil.ReadAll(r.Body)
		log.Info(string(body))
		var payload Payload
		var func_call FuncCall
		json.Unmarshal([]byte(body), &func_call)

		payload.Src = v.Image
		payload.Params = func_call.Params
		payload.Worker = func_call.Worker
		payload.Lang = func_call.Lang
		packet, marshal_err := json.Marshal(payload)
		client := http.Client{
			Timeout: 5 * time.Second,
		}
		var a_job Job
		a_job.payload = payload
		a_job.response_writer = w
		log.Info("Proxy: params inside job payload: " + a_job.payload.Params)
		job_queue.Add(a_job)

		resp, err := client.Post(payload.Worker, "application/json",
			bytes.NewBuffer(packet))

		if err != nil ||  marshal_err != nil {
			// log.Fatal(err)
			log.Info("HIT AN ERROR HERE ${err}")
			return
		}
		resp_body, _ := ioutil.ReadAll(resp.Body)
		log.Info(string(resp_body))


		hostName, _ := os.Hostname()
		d := &response{
			Function:     name,
			ResponseBody: string(resp_body) ,
			HostName:     hostName,
		}


//		responseBody, res_err := json.Marshal(d)
		_, res_err := json.Marshal(d)
		if res_err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			log.Errorf("error invoking %s. %v", name, err)
			return
		}

//		w.Write(responseBody)

		log.Info("!!!!!proxy request: %s completed.", name)
	}
}
