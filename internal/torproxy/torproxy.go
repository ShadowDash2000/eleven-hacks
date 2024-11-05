package torproxy

import (
	"context"
	"github.com/cretz/bine/control"
	"os"
	"time"

	"github.com/cretz/bine/process"
	"github.com/cretz/bine/tor"
	"github.com/pkg/errors"
)

type TorProxy struct {
	Tor   *tor.Tor
	Onion *tor.OnionService
}

func NewTorProxy(bridge string) (*TorProxy, error) {
	var args []string

	if bridge != "" {
		args = append(args, []string{
			"UseBridges", "1",
			"bridge", bridge,
			"ClientTransportPlugin", "obfs4 exec C:\\Users\\scout\\Desktop\\Tor Browser\\Browser\\TorBrowser\\Tor\\PluggableTransports\\lyrebird.exe",
		}...)
	}

	tmpPath := "tmp"
	err := os.MkdirAll(tmpPath, os.ModePerm)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create tmp directory")
	}

	t, err := tor.Start(nil, &tor.StartConf{
		ProcessCreator:  process.NewCreator("C:\\Users\\scout\\Desktop\\Tor Browser\\Browser\\TorBrowser\\Tor\\tor.exe"),
		TempDataDirBase: tmpPath,
		ExtraArgs:       args,
		DebugWriter:     nil,
	})
	if err != nil {
		return nil, errors.WithMessage(err, "Unable to start Tor")
	}

	ctx, _ := context.WithTimeout(context.Background(), 3*time.Minute)
	onion, err := t.Listen(ctx, &tor.ListenConf{
		RemotePorts: []int{80},
		Version3:    true,
	})
	if err != nil {
		return nil, errors.WithMessage(err, "Unable to create onion service")
	}

	return &TorProxy{
		Tor:   t,
		Onion: onion,
	}, nil
}

func (tp *TorProxy) Close() error {
	err := tp.Onion.Close()
	if err != nil {
		return errors.WithMessage(err, "Unable to close onion service")
	}

	err = tp.Tor.Close()
	if err != nil {
		return errors.WithMessage(err, "Unable to close Tor")
	}

	return nil
}

func (tp *TorProxy) SwapChain() (*control.Response, error) {
	res, err := tp.Tor.Control.SendRequest("SIGNAL NEWNYM")
	if err != nil {
		return nil, err
	}

	return res, err
}

func (tp *TorProxy) GetProxyAddress() (string, error) {
	info, err := tp.Tor.Control.GetInfo("net/listeners/socks")
	if err != nil {
		return "", errors.WithMessage(err, "Unable to get control info")
	}
	if len(info) != 1 || info[0].Key != "net/listeners/socks" {
		return "", errors.WithMessage(err, "Unexpected control info")
	}

	return info[0].Val, err
}
