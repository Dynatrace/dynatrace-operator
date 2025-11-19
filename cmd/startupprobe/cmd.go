package startupprobe

import (
	"context"
	"net"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	hostname       = "kubernetes.default.svc"
	use            = "startup-probe"
	defaultTimeout = 5
)

var (
	timeoutFlagValue = 5
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:  use,
		Long: "query DNS server about " + hostname,
		RunE: run,
	}

	cmd.PersistentFlags().IntVar(&timeoutFlagValue, "timeout", defaultTimeout, "specify a different timeout [s]")

	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	return cmd
}

func run(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutFlagValue)*time.Second)
	defer cancel()

	ips, err := net.DefaultResolver.LookupHost(ctx, hostname)
	if err != nil {
		return errors.WithMessagef(err, "DNS service not ready")
	}

	if len(ips) == 0 {
		return errors.Errorf("no DNS record found for %s", hostname)
	}

	return nil
}
