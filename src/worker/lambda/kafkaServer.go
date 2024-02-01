// ///// DRAFT CODE -- TO BE UPDATED /////

// package lambda

// import (
//     "errors"
//     "time"
//     "fmt"
//     "context"
// 	pb "path_to_be_updated/invocation.proto"
// 	lf "github.com/open-lambda/open-lambda/src/worker/lambda/lambdaFunction.go"

//     "github.com/segmentio/kafka-go"
//     "google.golang.org/protobuf/encoding/prototext"
// )

// // kafka-go repo: https://github.com/segmentio/kafka-go
// // kafka-go tutorial: https://www.sohamkamani.com/golang/working-with-kafka/

// const (
//     brokerAddress string = "localhost:9092"
//     topic string = "lambda-invocations"
// )

// // TODO: This struct to be completed
// type LambdaFunc struct {
// 	lmgr *LambdaMgr
// 	name string `protobuf:"bytes,1,opt,name=LambdaName"`

// 	rtType common.RuntimeType

// 	// lambda code
// 	lastPull *time.Time
// 	codeDir  string `protobuf:"bytes,2,opt,name=LambdaCodeDir"`
// 	meta     *sandbox.SandboxMeta

// 	// lambda execution
// 	funcChan  chan *Invocation // server to func
// 	instChan  chan *Invocation // func to instances
// 	doneChan  chan *Invocation // instances to func
// 	instances *list.List

// 	// send chan to the kill chan to destroy the instance, then
// 	// wait for msg on sent chan to block until it is done
// 	killChan chan chan bool
// }

// func invokeFunc(lambdaName string, lambdaCodeDir string) {
// 	// TODO: call invoke() inside lambdaFunction.go
// }

// var invoc Invocation

// func consume() {
//         fmt.Println("calling consume()")
//         r := kafka.NewReader(kafka.ReaderConfig{
//                 Brokers: []string{brokerAddress},
//                 Topic:   topic,
//                 GroupID: "consumer-group",
//                 MaxWait: 0.1 * time.Second,
//                 StartOffset: kafka.FirstOffset
//         })
//         for {
//                 protoMsg, readErr := r.ReadMessage(context.Background()) // the `ReadMessage` method blocks until we receive the next event
//                 if readErr != nil {
//                         panic("could not read message " + readErr.Error())
//                 }
//                 umErr := prototext.Unmarshal(protoMsg, &invoc)
//                 if umErr != nil {
//                         panic("proto message Unmarshal error" + umErr.Error())
//                 }
//                 invokeFunc(invoc.LambdaName, invoc.lambdaCodeDir)
//         }

//         if err := r.Close(); err != nil {
//                 log.Fatal("failed to close reader:", err)
//         }
// }
