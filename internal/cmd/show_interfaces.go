package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ravinald/nbcli/internal/cmdutils"
	"github.com/ravinald/nbcli/internal/netbox"
	"github.com/ravinald/nbcli/internal/output"
)

// interfaceKeywords is the positional keyword set for `nbcli show interfaces`.
//
// Filters: name, device, type, enabled, mgmt_only. `device` is the common
// narrow — Netbox interface lists are massive without it.
var interfaceKeywords = append([]cmdutils.KeywordSpec{
	{Name: "name", Description: "exact interface name"},
	{Name: "device", Description: "device name (highly recommended)"},
	{Name: "type", Description: "interface type", Example: "10gbase-x-sfpp"},
	{Name: "enabled", Description: "true|false",
		Values: []string{"true", "false"}},
	{Name: "mgmt_only", Description: "true|false",
		Values: []string{"true", "false"}},
}, cmdutils.PaginationKeywords()...)

func newShowInterfacesCmd(io IO) *cobra.Command {
	return &cobra.Command{
		Use:   "interfaces " + cmdutils.UsageLine(interfaceKeywords),
		Short: "List DCIM interfaces",
		Long: "List DCIM interfaces. Filters are positional keyword/value pairs.\n\n" +
			cmdutils.HelpTable(interfaceKeywords) +
			"\nExamples:\n" +
			"  nbcli show interfaces device hq-sw-01\n" +
			"  nbcli show interfaces device hq-sw-01 enabled true\n" +
			"  nbcli show interfaces device hq-sw-01 mgmt_only true\n",
		Args:              cmdutils.Validator(interfaceKeywords),
		ValidArgsFunction: cmdutils.CompletionFunc(interfaceKeywords),
		RunE: func(cmd *cobra.Command, args []string) error {
			kv, _ := cmdutils.ParseShowArgs(args, interfaceKeywords)

			opts := netbox.ListInterfacesOptions{
				Name:   kv["name"],
				Device: kv["device"],
				Type:   kv["type"],
				Limit:  50,
			}
			if v, ok := kv["enabled"]; ok {
				b, err := strconv.ParseBool(v)
				if err != nil {
					return fmt.Errorf("enabled must be true or false: %w", err)
				}
				opts.Enabled = &b
			}
			if v, ok := kv["mgmt_only"]; ok {
				b, err := strconv.ParseBool(v)
				if err != nil {
					return fmt.Errorf("mgmt_only must be true or false: %w", err)
				}
				opts.MgmtOnly = &b
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
				{Header: "ID", Extract: func(r any) string { return strconv.Itoa(r.(netbox.Interface).ID) }},
				{Header: "Name", Extract: func(r any) string { return r.(netbox.Interface).Name }},
				{Header: "Device", Extract: func(r any) string {
					if r.(netbox.Interface).Device == nil {
						return ""
					}
					return r.(netbox.Interface).Device.Name
				}},
				{Header: "Type", Extract: func(r any) string { return r.(netbox.Interface).Type.Label }},
				{Header: "Enabled", Extract: func(r any) string {
					return strconv.FormatBool(r.(netbox.Interface).Enabled)
				}},
				{Header: "MAC", Extract: func(r any) string { return r.(netbox.Interface).MACAddress }},
				{Header: "MTU", Extract: func(r any) string {
					if m := r.(netbox.Interface).MTU; m != nil {
						return strconv.Itoa(*m)
					}
					return ""
				}},
				{Header: "MgmtOnly", Extract: func(r any) string {
					return strconv.FormatBool(r.(netbox.Interface).MgmtOnly)
				}},
			}
			iterOpts := netbox.IterateOptions{PageSize: 100, MaxPages: 200}

			if fetchAll {
				return renderStreaming[netbox.Interface](cmd, io, client.InterfacesFetcher(opts), iterOpts, cols)
			}
			page, err := client.ListInterfaces(cmd.Context(), opts)
			if err != nil {
				return err
			}
			return renderRows(cmd, io, page.Results, cols)
		},
	}
}
