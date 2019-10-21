package explorer

import (
	"errors"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type ExplorerAvailablePerformance struct {
	BaseExplorer

	clusterID string
}

func (this *ExplorerAvailablePerformance) Construct(timerNotyfy time.Duration) *ExplorerAvailablePerformance {
	this.summary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "AvailablePerformance",
			Help: "Доступная производительность хоста",
		},
		[]string{"host"},
	)

	this.timerNotyfy = timerNotyfy
	prometheus.MustRegister(this.summary)
	return this
}

func (this *ExplorerAvailablePerformance) StartExplore() {
	t := time.NewTicker(this.timerNotyfy)
	for {
		if licCount, err := this.getData(); err == nil {
			for key, value := range licCount {
				this.summary.WithLabelValues(key).Observe(value)
			}
		} else {
			this.summary.WithLabelValues("").Observe(0) // Для того что бы в ответе был AvailablePerformance, нужно дл атотестов
			log.Println("Произошла ошибка: ", err.Error())
		}

		<-t.C
	}
}

func (this *ExplorerAvailablePerformance) getData() (data map[string]float64, err error) {
	data = make(map[string]float64)

	if this.clusterID == "" {
		cmdCommand := exec.Command("/opt/1C/v8.3/x86_64/rac", "cluster", "list") // TODO: вынести путь к rac в конфиг

		cluster := make(map[string]string)
		if result, e := this.run(cmdCommand); e != nil {
			fmt.Println("Произошла ошибка выполнения: ", e.Error())
			return data, e
		} else {
			cluster = this.formatResult(result)
		}

		if id, ok := cluster["cluster"]; !ok {
			err = errors.New("Не удалось получить идентификатор кластера")
			return data, err
		} else {
			this.clusterID = id
		}
	}

	// /opt/1C/v8.3/x86_64/rac process --cluster=ee5adb9a-14fa-11e9-7589-005056032522 list
	procData := []map[string]string{}

	param := []string{}
	param = append(param, "process")
	param = append(param, "list")
	param = append(param, fmt.Sprintf("--cluster=%v", this.clusterID))

	cmdCommand := exec.Command("/opt/1C/v8.3/x86_64/rac", param...)
	if result, err := this.run(cmdCommand); err != nil {
		log.Println("Произошла ошибка выполнения: ", err.Error())
		return data, err
	} else {
		this.formatMultiResult(result, &procData)
	}

	// У одного хоста может быть несколько рабочих процессов в таком случаи мы берем среднее арифметическое по процессам
	tmp := make(map[string][]int)
	for _, item := range procData {
		if perfomance, err := strconv.Atoi(item["available-perfomance"]); err == nil {
			tmp[item["host"]] = append(tmp[item["host"]], perfomance)
		}
	}
	for key, value := range tmp {
		for _, item := range value {
			data[key] += float64(item)
		}
		data[key] = data[key] / float64(len(value))
	}
	return data, nil
}

func (this *ExplorerAvailablePerformance) formatMultiResult(data string, licData *[]map[string]string) {
	reg := regexp.MustCompile(`(?m)^$`)
	for _, part := range reg.Split(data, -1) {
		*licData = append(*licData, this.formatResult(part))
	}
}

func (this *ExplorerAvailablePerformance) formatResult(strIn string) map[string]string {
	result := make(map[string]string)

	for _, line := range strings.Split(strIn, "\n") {
		parts := strings.Split(line, ":")
		if len(parts) == 2 {
			result[strings.Trim(parts[0], " ")] = strings.Trim(parts[1], " ")
		}
	}

	return result
}

func (this *ExplorerAvailablePerformance) GetName() string {
	return "aperf"
}
