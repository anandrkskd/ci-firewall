package requestor

import (
	"encoding/json"
	"fmt"

	"github.com/mohammedzee1000/ci-firewall/pkg/messages"
	"github.com/mohammedzee1000/ci-firewall/pkg/queue"
	"github.com/streadway/amqp"
)

type Requestor struct {
	sendq            *queue.AMQPQueue
	rcvq             *queue.AMQPQueue
	jenkinsProject   string
	jenkinsToken     string
	jenkinsBuild     int
	repoURL          string
	kind             string
	target           string
	runscript        string
	recieveQueueName string
	done             chan error
}

func NewRequestor(amqpURI, sendqName, jenkinsProject, jenkinsToken, repoURL, kind, target, runscript, recieveQueueName string) *Requestor {
	r := &Requestor{
		sendq:            queue.NewAMQPQueue(amqpURI, sendqName),
		rcvq:             queue.NewAMQPQueue(amqpURI, recieveQueueName),
		jenkinsProject:   jenkinsProject,
		jenkinsToken:     jenkinsToken,
		jenkinsBuild:     -1,
		repoURL:          repoURL,
		kind:             kind,
		target:           target,
		runscript:        runscript,
		recieveQueueName: recieveQueueName,
		done:             make(chan error),
	}
	return r
}

func (r *Requestor) initQueus() error {
	if r.kind != messages.RequestTypePR && r.kind != messages.RequestTypeBranch && r.kind != messages.RequestTypeTag {
		return fmt.Errorf("kind should be %s, %s or %s", messages.RequestTypePR, messages.RequestTypeBranch, messages.RequestTypeTag)
	}
	err := r.sendq.Init()
	if err != nil {
		return fmt.Errorf("failed to initalize send q %w", err)
	}
	err = r.rcvq.Init()
	if err != nil {
		return fmt.Errorf("failed to initialize rcvq %w", err)
	}
	return nil
}

func (r *Requestor) sendBuildRequest() error {
	var err error
	err = r.sendq.Publish(true, messages.NewRemoteBuildRequestMessage(r.jenkinsProject, r.jenkinsToken, r.repoURL, r.kind, r.target, r.runscript, r.recieveQueueName))
	if err != nil {
		return fmt.Errorf("failed to send build request %w", err)
	}
	return nil
}

func (r *Requestor) consumeMessages() error {
	err := r.rcvq.Consume(func(deliveries <-chan amqp.Delivery, done chan error) {
		for d := range deliveries {
			m := &messages.Message{}
			err1 := json.Unmarshal(d.Body, m)
			if err1 != nil {
				done <- fmt.Errorf("failed to unmarshal as message %w", err1)
				return
			}
			if r.jenkinsBuild == -1 && m.IsBuild() {
				bm := messages.NewBuildMessage(-1)
				err1 = json.Unmarshal(d.Body, bm)
				if err1 != nil {
					done <- fmt.Errorf("failed to unmarshal as build message %w", err1)
					return
				}
				r.jenkinsBuild = bm.Build
			} else if r.jenkinsBuild == m.Build {
				if m.ISLog() {
					lm := messages.NewLogsMessage(-1, "")
					err1 = json.Unmarshal(d.Body, lm)
					if err1 != nil {
						done <- fmt.Errorf("failed to unmarshal as logs message %w", err1)
						return
					}
					fmt.Println(lm.Logs)
				} else if m.IsStatus() {
					sm := messages.NewStatusMessage(-1, false)
					err1 = json.Unmarshal(d.Body, sm)
					if err1 != nil {
						done <- fmt.Errorf("failed to unmarshal as status message %w", err1)
					}
					if !sm.Success {
						done <- fmt.Errorf("Failed the test, see logs above ^")
						return
					}
					done <- nil
					return
				}
			}
			d.Ack(false)
		}
	}, r.done)
	if err != nil {
		return err
	}
	return nil
}

func (r *Requestor) Run() error {
	err := r.initQueus()
	if err != nil {
		return err
	}
	err = r.sendBuildRequest()
	if err != nil {
		return err
	}
	err = r.consumeMessages()
	if err != nil {
		return err
	}
	return nil
}

func (r *Requestor) Done() chan error {
	return r.done
}

func (r *Requestor) ShutDown() error {
	err := r.sendq.Shutdown()
	if err != nil {
		return fmt.Errorf("failed to shutdown send q %w", err)
	}
	err = r.rcvq.Shutdown()
	if err != nil {
		return fmt.Errorf("failed to shutdown rcv q %w", err)
	}
	return nil
}