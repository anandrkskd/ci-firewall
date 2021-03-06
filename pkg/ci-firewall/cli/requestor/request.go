package requestor

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mohammedzee1000/ci-firewall/pkg/ci-firewall/cli/genericclioptions"
	"github.com/mohammedzee1000/ci-firewall/pkg/jenkins"
	"github.com/mohammedzee1000/ci-firewall/pkg/messages"
	"github.com/mohammedzee1000/ci-firewall/pkg/requestor"
	"github.com/spf13/cobra"
)

const RequestRecommendedCommandName = "request"

type RequestOptions struct {
	requestor        *requestor.Requestor
	amqpURI          string
	sendQName        string
	sendExchangeName string
	sendTopic        string
	repoURL          string
	kind             string
	target           string
	runScript        string
	setupScript      string
	rcvIdent         string
	jenkinsproject   string
	runScriptURL     string
	mainBranch       string
	timeout          time.Duration
	lazy             bool
}

func NewRequestOptions() *RequestOptions {
	return &RequestOptions{}
}

func (ro *RequestOptions) Complete(name string, cmd *cobra.Command, args []string) error {
	if ro.kind == "" {
		ro.kind = messages.RequestTypePR
	}
	if ro.rcvIdent == "" {
		ro.rcvIdent = fmt.Sprintf("amqp.ci.rcv.%s.%s.%s", ro.jenkinsproject, ro.kind, ro.target)
	}
	if ro.lazy {
		ro.rcvIdent = fmt.Sprintf("%s.lazy", ro.rcvIdent)
	}
	return nil
}

func (ro *RequestOptions) Validate() (err error) {
	if ro.amqpURI == "" {
		return fmt.Errorf("provide AMQP URI")
	}
	if ro.sendExchangeName == "" {
		return fmt.Errorf("please provide send exchange name")
	}
	if ro.sendTopic == "" {
		return fmt.Errorf("please provide send q topic")
	}
	if ro.repoURL == "" {
		return fmt.Errorf("provide Repo URL")
	}
	if ro.kind == "" {
		return fmt.Errorf("provide Kind")
	}
	if ro.target == "" {
		return fmt.Errorf("provide Target")
	}
	if ro.runScript == "" {
		return fmt.Errorf("provide Run Script")
	}
	if ro.kind != messages.RequestTypePR && ro.kind != messages.RequestTypeBranch && ro.kind != messages.RequestTypeTag {
		return fmt.Errorf("kind must be one of these 3 %s|%s|%s", messages.RequestTypePR, messages.RequestTypeBranch, messages.RequestTypeTag)
	}
	if ro.kind == messages.RequestTypePR && ro.mainBranch == "" {
		return fmt.Errorf("main branch must be provided if kind is pr")
	}
	return nil
}

func (ro *RequestOptions) Run() (err error) {
	ro.requestor = requestor.NewRequestor(
		ro.amqpURI,
		ro.sendQName,
		ro.sendExchangeName,
		ro.sendTopic,
		ro.repoURL,
		ro.kind,
		ro.target,
		ro.setupScript,
		ro.runScript,
		ro.rcvIdent,
		ro.runScriptURL,
		ro.mainBranch,
	)
	err = ro.requestor.Run()
	if err != nil {
		return err
	}
	select {
	case done := <-ro.requestor.Done():
		if done == nil {
			log.Println("Tests succeeeded, see logs above ^")
			if err := ro.requestor.ShutDown(); err != nil {
				return fmt.Errorf("error during shutdown: %w", err)
			}
		} else {
			if err := ro.requestor.ShutDown(); err != nil {
				return fmt.Errorf("error during shutdown: %w", err)
			}
			return fmt.Errorf("failed due to err %w", done)
		}
	case <-time.After(ro.timeout):
		if err := ro.requestor.ShutDown(); err != nil {
			return fmt.Errorf("error during shutdown: %w", err)
		}
		return fmt.Errorf("timed out")
	}
	return nil
}

func NewCmdRequestor(name, fullname string) *cobra.Command {
	o := NewRequestOptions()
	cmd := &cobra.Command{
		Use:   name,
		Short: "request a build",
		Run: func(cmd *cobra.Command, args []string) {
			genericclioptions.GenericRun(o, cmd, args)
		},
	}
	cmd.Flags().StringVar(&o.amqpURI, "amqpuri", os.Getenv("AMQP_URI"), "the url of amqp server")
	cmd.Flags().StringVar(&o.jenkinsproject, "jenkinsproject", jenkins.GetJenkinsJob(), "the name of target jenkins project. Required for ident purposes only")
	cmd.Flags().StringVar(&o.sendQName, "sendqueue", "amqp.ci.queue.send", "the name of the send queue")
	cmd.Flags().StringVar(&o.sendExchangeName, "sendexchange", "amqp.ci.exchange.send", "the name of the exchange tp use for send")
	cmd.Flags().StringVar(&o.sendTopic, "sendtopic", "amqp.ci.topic.send", "the name of the send topic")
	cmd.Flags().StringVar(&o.rcvIdent, "rcvident", os.Getenv(messages.RequestParameterRcvQueueName), "the name of the recieve queue")
	cmd.Flags().StringVar(&o.repoURL, "repourl", os.Getenv(messages.RequesParameterRepoURL), "the url of the repo to clone on jenkins")
	cmd.Flags().StringVar(&o.kind, "kind", os.Getenv(messages.RequestParameterKind), "the kind of build you want to do")
	cmd.Flags().StringVar(&o.target, "target", os.Getenv(messages.RequestParameterTarget), "the target is based on kind. Can be pr no or branch name or tag name")
	cmd.Flags().StringVar(&o.runScript, "runscript", os.Getenv(messages.RequestParameterRunScript), "the path of the script to run on jenkins, relative to repo root")
	cmd.Flags().StringVar(&o.runScriptURL, "runscripturl", "", "the url of remote run script, if any. Must be providede with --runscript as that is what it will be downloaded as")
	cmd.Flags().StringVar(&o.setupScript, "setupscript", os.Getenv(messages.RequestParameterSetupScript), "the setup script to run")
	cmd.Flags().DurationVar(&o.timeout, "timeout", 15*time.Minute, "timeout duration ")
	cmd.Flags().StringVar(&o.mainBranch, "mainbranch", "master", "the main branch, to be provided if kind is PR")
	cmd.Flags().BoolVar(&o.lazy, "lazy", false, "Use lazy queues. This simply appends lazy to rcv queue name. So configure rabbitmq accordingly. see https://www.rabbitmq.com/lazy-queues.html")
	return cmd
}
