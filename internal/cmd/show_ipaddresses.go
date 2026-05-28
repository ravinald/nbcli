package cmd

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ravinald/nbcli/internal/cmdutils"
	"github.com/ravinald/nbcli/internal/netbox"
	"github.com/ravinald/nbcli/internal/output"
)

var ipAddressKeywords = append([]cmdutils.KeywordSpec{
	{Name: "address", Description: "exact IP/CIDR"},
	{Name: "parent", Description: "containing prefix (CIDR)"},
	{Name: "vrf", Description: "VRF RD"},
	{Name: "family", Description: "4 or 6", Values: []string{"4", "6"}},
	{Name: "status", Description: "status value",
		Values: []string{"active", "reserved", "deprecated", "dhcp", "slaac"}},
	{Name: "role", Description: "role value", Example: "loopback"},
	{Name: "tenant", Description: "tenant slug"},
	{Name: "device", Description: "assigned device name"},
	{Name: "vm", Description: "assigned VM name"},
}, cmdutils.PaginationKeywords()...)

func newShowIPAddressesCmd(io IO) *cobra.Command {
	return &cobra.Command{
		Use:   "ip-addresses " + cmdutils.UsageLine(ipAddressKeywords),
		Short: "List IPAM IP addresses",
		Long: "List IPAM IP addresses. Filters are positional keyword/value pairs.\n\n" +
			cmdutils.HelpTable(ipAddressKeywords) +
			"\nExamples:\n" +
			"  nbcli show ip-addresses parent 10.0.0.0/24\n" +
			"  nbcli show ip-addresses device hq-sw-01\n",
		Aliases:           []string{"ips", "ip"},
		Args:              cmdutils.Validator(ipAddressKeywords),
		ValidArgsFunction: cmdutils.CompletionFunc(ipAddressKeywords),
		RunE: func(cmd *cobra.Command, args []string) error {
			kv, _ := cmdutils.ParseShowArgs(args, ipAddressKeywords)
			opts := netbox.ListIPAddressesOptions{
				Address: kv["address"],
				Parent:  kv["parent"],
				VRF:     kv["vrf"],
				Family:  kv["family"],
				Status:  kv["status"],
				Role:    kv["role"],
				Tenant:  kv["tenant"],
				Device:  kv["device"],
				VM:      kv["vm"],
				Limit:   50,
			}
			fetchAll, err := cmdutils.ApplyLimitOffset(kv, &opts.Limit, &opts.Offset)
			if err != nil {
				return err
			}
			client, err := clientFromCtx(cmd)
			if err != nil {
				return err
			}
			cols := []output.Column{
				{Header: "ID", Extract: func(r any) string { return strconv.Itoa(r.(netbox.IPAddress).ID) }},
				{Header: "Address", Extract: func(r any) string { return r.(netbox.IPAddress).Address }},
				{Header: "Family", Extract: func(r any) string { return r.(netbox.IPAddress).Family.Label }},
				{Header: "VRF", Extract: func(r any) string {
					if r.(netbox.IPAddress).VRF == nil {
						return ""
					}
					return r.(netbox.IPAddress).VRF.Name
				}},
				{Header: "Status", Extract: func(r any) string { return r.(netbox.IPAddress).Status.Label }},
				{Header: "DNS", Extract: func(r any) string { return r.(netbox.IPAddress).DNSName }},
				{Header: "Tenant", Extract: func(r any) string {
					if r.(netbox.IPAddress).Tenant == nil {
						return ""
					}
					return r.(netbox.IPAddress).Tenant.Name
				}},
			}
			iterOpts := netbox.IterateOptions{PageSize: 100, MaxPages: 200}

			if fetchAll {
				return renderStreaming[netbox.IPAddress](cmd, io, client.IPAddressesFetcher(opts), iterOpts, cols)
			}
			page, err := client.ListIPAddresses(cmd.Context(), opts)
			if err != nil {
				return err
			}
			return renderRows(cmd, io, page.Results, cols)
		},
	}
}
