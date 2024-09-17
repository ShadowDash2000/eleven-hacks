package chromeproxy

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/pkg/errors"
)

type ChromeProxy struct {
	ProxyAddress string
}

func NewChromeProxy(proxyAddress string) *ChromeProxy {
	return &ChromeProxy{
		ProxyAddress: proxyAddress,
	}
}

func (cp *ChromeProxy) NewExecAllocator(headless bool) (context.Context, context.CancelFunc) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ProxyServer(fmt.Sprintf("socks5://%s", cp.ProxyAddress)),
		chromedp.Flag("headless", headless),
	)
	return chromedp.NewExecAllocator(context.Background(), opts...)
}

// StartCaptchaSolve
//
// Starts a new Chrome exec with proxy and head and navigating to
// ElevenLabs sign-up page to manually solve captcha.
// Every second checking for data-hcaptcha-response attribute.
// After the captcha was solved returns captcha response.
func (cp *ChromeProxy) StartCaptchaSolve() (string, error) {
	ctx, cancel := cp.NewExecAllocator(false)
	defer cancel()
	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	var statusCode int
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if ev, ok := ev.(*network.EventResponseReceived); ok {
			statusCode = int(ev.Response.Status)
		}
	})

	err := chromedp.Run(ctx,
		chromedp.Navigate("https://elevenlabs.io/app/sign-up"),
	)
	if err != nil {
		return "", errors.WithMessage(err, "Unable to start captcha solve")
	}

	if statusCode != 200 {
		return "", errors.Errorf("Server returned status code %d", statusCode)
	}

	tasks := chromedp.Tasks{
		chromedp.WaitReady("form", chromedp.ByQuery),
		chromedp.Evaluate(`
		let termsCheckbox = document.querySelector('input[name="terms"]');
		termsCheckbox?.previousSibling.click();
		`, nil),
		chromedp.SendKeys("input[name=\"email\"]", "trigger-the-fuckin@captcha.com", chromedp.ByQuery),
		chromedp.SendKeys("input[name=\"password\"]", "SW@iqU8g.B?fW9p", chromedp.ByQuery),
		chromedp.Evaluate(`
		let submitButton = document.querySelector('form button[data-testid="signup-signup-button-div"]');
		submitButton?.click();
		`, nil),
	}

	err = chromedp.Run(ctx, tasks)
	if err != nil {
		return "", err
	}

	var captchaResponse *string
	for {
		chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
			node, err := dom.GetDocument().Do(ctx)
			if err != nil {
				return err
			}

			nodeId, err := dom.QuerySelector(node.NodeID, "iframe[data-hcaptcha-widget-id]").Do(ctx)
			if err != nil {
				return err
			}

			attrs, err := dom.GetAttributes(nodeId).Do(ctx)
			if err != nil {
				return err
			}

			next := false
			for _, attr := range attrs {
				if next {
					if attr == "" {
						break
					}
					captchaResponse = &attr
					break
				}
				if attr == "data-hcaptcha-response" {
					next = true
				}
			}

			return nil
		}))

		if captchaResponse != nil {
			break
		}

		time.Sleep(500 * time.Millisecond)
	}

	return *captchaResponse, nil
}
