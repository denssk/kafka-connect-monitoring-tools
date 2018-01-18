package main

import (
  "io"
  "net/http"
  "log"
  "os"
  "fmt"
  "time"
  "strconv"
  "github.com/jesseadams/kafka-connect-monitoring-tools/common"
)

var ResponseString = "Initializing..."

func RetrieveKafkaConnectMetrics(hostString string) string {
  var output string

  connectors, err := kafka_connect.ListConnectors(hostString, true)
  if err != nil {
    fmt.Println(err)
  }

  connectorCount := len(connectors)
  output += "# TYPE kafka_connect_connectorcount gauge\n"
  line := fmt.Sprintf("kafka_connect_connectorcount %.1f\n", float64(connectorCount))
  output += line

  output += "# TYPE kafka_connect_runningtaskscount gauge\n"
  for _, connector := range connectors {
      status := new(kafka_connect.KafkaConnectorStatus)
      err = kafka_connect.CheckStatus(hostString, connector, status, true)

      if err != nil {
        fmt.Println(err)
      }

      failedTasksCount := 0.0
      for _, task := range status.Tasks {
        if task.State == "FAILED" {
          failedTasksCount += 1.0
        }
      }
      line := fmt.Sprintf("kafka_connect_failed_tasks_count{connector=\"%s\"} %.1f\n", connector, failedTasksCount)
      output += line
  }

  return output
}

func PublishPrometheusMetrics(writer http.ResponseWriter, req *http.Request) {
  io.WriteString(writer, ResponseString)
}

func RetrievePrometheusMetrics() {
  kafkaConnectHostString := os.Getenv("KAFKA_CONNECT_URL")
  schemaRegistryHostString := os.Getenv("SCHEMA_REGISTRY_URL")
  metricsRefreshRate, err := strconv.ParseInt(os.Getenv("METRICS_REFRESH_RATE"), 10, 32)

  if err != nil {
    fmt.Println("Unable to parse refresh interval from METRICS_REFRESH_RATE")
    fmt.Println(err)
  }

  if metricsRefreshRate < 10 {
    metricsRefreshRate = 60
  }

  fmt.Printf("Metrics Refresh Rate: %d seconds\n", metricsRefreshRate)
  for true {
    fmt.Println("Refreshing metrics...")
    kafkaConnectOutput := RetrieveKafkaConnectMetrics(kafkaConnectHostString)
    ResponseString = kafkaConnectOutput
    fmt.Println("Metrics refresh complete!")

    time.Sleep(time.Duration(metricsRefreshRate) * time.Second)
  }
}

func main() {
  go RetrievePrometheusMetrics()
  http.HandleFunc("/metrics", PublishPrometheusMetrics)
  log.Fatal(http.ListenAndServe(":7071", nil))
}
