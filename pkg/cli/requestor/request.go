package requestor

import (
	"fmt"
	"log"
	"time"

	"github.com/mohammedzee1000/ci-firewall/pkg/cli/genericclioptions"
	"github.com/mohammedzee1000/ci-firewall/pkg/messages"
	"github.com/mohammedzee1000/ci-firewall/pkg/requestor"
	"github.com/spf13/cobra"
)

const RequestRecommendedCommandName = "request"

type RequestOptions struct {
	requestor      *requestor.Requestor
	amqpURI        string
	sendQName      string
	jenkinsProject string
	jenkinsToken   string
	repoURL        string
	kind           string
	target         string
	runScript      string
	recieveQName   string
}

func NewRequestOptions() *RequestOptions {
	return &RequestOptions{}
}

func (ro *RequestOptions) Complete(name string, cmd *cobra.Command, args []string) error {
	if ro.sendQName == "" {
		ro.sendQName = "CI_SEND"
	}
	if ro.recieveQName == "" {
		ro.recieveQName = fmt.Sprintf("rcv_%s_%s", ro.jenkinsProject, ro.target)
	}
	if ro.kind == "" {
		ro.kind = messages.RequestTypePR
	}
	return nil
}

func (ro *RequestOptions) Validate() (err error) {
	if ro.amqpURI == "" {
		return fmt.Errorf("provide AMQP URI")
	}
	if ro.jenkinsProject == "" {
		return fmt.Errorf("provide Jenkins Project")
	}
	if ro.jenkinsToken == "" {
		return fmt.Errorf("provide Jenkins Token")
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
	return nil
}

func (ro *RequestOptions) Run() (err error) {
	ro.requestor = requestor.NewRequestor(
		ro.amqpURI,
		ro.sendQName,
		ro.jenkinsProject,
		ro.jenkinsToken,
		ro.repoURL,
		ro.kind,
		ro.target,
		ro.runScript,
		ro.recieveQName,
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
			return fmt.Errorf("failed due to err %w", err)
		}
	case <-time.After(12 * time.Minute):
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
	cmd.Flags().StringVar(&o.amqpURI, "amqpurl", "", "the url of amqp server")
	cmd.Flags().StringVar(&o.sendQName, "sendqueue", "", "the name of the send queue")
	cmd.Flags().StringVar(&o.recieveQName, "recievequeue", "", "the name of the recieve queue")
	cmd.Flags().StringVar(&o.jenkinsProject, "jenkinsproject", "", "the name of the jenkins project")
	cmd.Flags().StringVar(&o.jenkinsToken, "jenkinstoken", "", "the token as set on jenkins project for remote build")
	cmd.Flags().StringVar(&o.repoURL, "repourl", "", "the url of the repo to clone on jenkins")
	cmd.Flags().StringVar(&o.kind, "kind", "", "the kind of build you want to do")
	cmd.Flags().StringVar(&o.target, "target", "", "the target is based on kind. Can be pr no or branch name or tag name")
	cmd.Flags().StringVar(&o.runScript, "run", "", "the path of the script to run on jenkins, relative to repo root")
	return cmd
}