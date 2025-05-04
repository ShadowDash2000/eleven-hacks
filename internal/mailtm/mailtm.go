package mailtm

import (
	"eleven-hacks/pkg/htmlcrawler"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/felixstrobel/mailtm"
	"github.com/gorilla/schema"
	"github.com/pkg/errors"
	"golang.org/x/net/html"
)

type MailTM struct {
	Client *mailtm.MailClient
}

type ConfirmationData struct {
	Mode         string `schema:"mode"`
	OobCode      string `schema:"oobCode"`
	ApiKey       string `schema:"apiKey"`
	Lang         string `schema:"lang"`
	InternalCode string `schema:"internalCode"`
	UserId       string `schema:"userId"`
	NewUser      bool   `schema:"newUser"`
}

func NewMailTM() (*MailTM, error) {
	client, err := mailtm.New()
	if err != nil {
		return nil, errors.WithMessage(err, "Unable to create MailTM client")
	}

	return &MailTM{
		Client: client,
	}, nil
}

func (mtm *MailTM) NewAccount() (*mailtm.Account, error) {
	return mtm.Client.NewAccount()
}

func (mtm *MailTM) DeleteAccount(account *mailtm.Account) error {
	return mtm.Client.DeleteAccount(account)
}

func (mtm *MailTM) GetLinkWithPrefix(message *mailtm.DetailedMessage, prefix string) (string, error) {
	for _, rawHtml := range message.Html {
		htmlReader := strings.NewReader(rawHtml)

		document, err := html.Parse(htmlReader)
		if err != nil {
			return "", err
		}

		nodes := htmlcrawler.CrawlByTagAll("a", document)
		for _, node := range nodes {
			attributes := htmlcrawler.GetNodeAttributes(node)
			if _, ok := attributes["href"]; !ok {
				continue
			}

			if ok := strings.HasPrefix(attributes["href"], prefix); ok {
				return attributes["href"], nil
			}
		}
	}

	return "", fmt.Errorf("Link with prefix %s not found in message", prefix)
}

func (mtm *MailTM) GetConfirmationData(rawUrl string) (*ConfirmationData, error) {
	parsedUrl, err := url.Parse(rawUrl)
	if err != nil {
		return nil, errors.WithMessage(err, "Unable to parse url")
	}

	parsedQuery, err := url.ParseQuery(parsedUrl.RawQuery)
	if err != nil {
		return nil, errors.WithMessage(err, "Unable to parse query")
	}

	decoder := schema.NewDecoder()
	var confirmationData ConfirmationData
	err = decoder.Decode(&confirmationData, parsedQuery)
	if err != nil {
		return nil, errors.WithMessage(err, "Unable to decode confirmation data")
	}

	return &confirmationData, nil
}

func (mtm *MailTM) WaitForConfirmationEmail(account *mailtm.Account, retryCount int) (*mailtm.DetailedMessage, error) {
	var message *mailtm.Message
	var err error
	sleepTime := time.Duration(1)
	for i := 0; i < retryCount; i++ {
		message, err = mtm.GetLastMessage(account)
		if err == nil {
			break
		}
		time.Sleep(sleepTime * time.Second)
	}

	if message == nil {
		return nil, errors.Errorf("Confirmation email not found after %d seconds", int(sleepTime)*retryCount)
	}

	return mtm.Client.GetMessageByID(account, message.ID)
}

func (mtm *MailTM) GetLastMessage(account *mailtm.Account) (*mailtm.Message, error) {
	messages, err := mtm.Client.GetMessages(account, 1)
	if err != nil {
		return nil, errors.WithMessage(err, "Unable to get messages")
	}

	if len(messages) == 0 {
		return nil, errors.New("No messages found")
	}

	return &messages[0], nil
}
