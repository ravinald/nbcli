package cmd

import (
	"github.com/spf13/cobra"

	"github.com/ravinald/nbcli/internal/cmdutils"
	"github.com/ravinald/nbcli/internal/netbox"
)

// deviceKeywords is the positional keyword set for `nbcli show devices`.
var deviceKeywords = append([]cmdutils.KeywordSpec{
	{Name: "name", Description: "exact device name"},
	{Name: "role", Description: "device role slug"},
	{Name: "site", Description: "site slug"},
	{Name: "rack", Description: "rack name"},
	{Name: "status", Description: "status value", Example: "active",
		Values: []string{"offline", "active", "planned", "staged", "failed", "inventory", "decommissioning"}},
	{Name: "tenant", Description: "tenant slug"},
	{Name: "manufacturer", Description: "manufacturer slug"},
	{Name: "model", Description: "device-type slug"},
	{Name: "location", Description: "location slug"},
	{Name: "tag", Description: "tag slug"},
}, cmdutils.PaginationKeywords()...)

func newShowDevicesCmd(io IO) *cobra.Command {
	return &cobra.Command{
		Use:   "devices " + cmdutils.UsageLine(deviceKeywords),
		Short: "List DCIM devices",
		Long: "List DCIM devices. Filters are positional keyword/value pairs.\n\n" +
			cmdutils.HelpTable(deviceKeywords) +
			"\nExamples:\n" +
			"  nbcli show devices site hq status active\n" +
			"  nbcli show devices manufacturer juniper model ex4300\n" +
			"  nbcli show devices role switch limit 0\n",
		Args:              cmdutils.Validator(deviceKeywords),
		ValidArgsFunction: cmdutils.CompletionFunc(deviceKeywords),
		RunE: func(cmd *cobra.Command, args []string) error {
			kv, _ := cmdutils.ParseShowArgs(args, deviceKeywords)

			opts := netbox.ListDevicesOptions{
				Name:         kv["name"],
				Role:         kv["role"],
				Site:         kv["site"],
				Rack:         kv["rack"],
				Status:       kv["status"],
				Tenant:       kv["tenant"],
				Manufacturer: kv["manufacturer"],
				Model:        kv["model"],
				Location:     kv["location"],
				Tag:          kv["tag"],
				Limit:        50,
			}
			fetchAll, err := cmdutils.ApplyLimitOffset(kv, &opts.Limit, &opts.Offset)
			if err != nil {
				return err
			}

			client, err := clientFromCtx(cmd)
			if err != nil {
				return err
			}

			cols := resolveColumns(cmd, "devices")
			iterOpts := netbox.IterateOptions{PageSize: 100, MaxPages: 200}

			if fetchAll {
				return renderStreaming[netbox.Device](cmd, io, client.DevicesFetcher(opts), iterOpts, cols)
			}
			page, err := client.ListDevices(cmd.Context(), opts)
			if err != nil {
				return err
			}
			return renderRows(cmd, io, page.Results, cols)
		},
	}
}
