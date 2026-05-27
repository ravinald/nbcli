package cmd

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ravinald/nbcli/internal/cmdutils"
	"github.com/ravinald/nbcli/internal/netbox"
	"github.com/ravinald/nbcli/internal/output"
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

			var rows []netbox.Device
			if fetchAll {
				rows, err = netbox.ListAll(cmd.Context(),
					client.DevicesFetcher(opts),
					netbox.IterateOptions{PageSize: 100, MaxPages: 200})
			} else {
				var page netbox.Page[netbox.Device]
				page, err = client.ListDevices(cmd.Context(), opts)
				rows = page.Results
			}
			if err != nil {
				return err
			}

			cfg := configFromCtx(cmd.Context())
			explicit, err := output.Parse(cfg.Format)
			if err != nil {
				return err
			}
			format := output.Resolve(explicit, io.Out)
			r, err := output.New(format)
			if err != nil {
				return err
			}
			cols := []output.Column{
				{Header: "ID", Extract: func(r any) string { return strconv.Itoa(r.(netbox.Device).ID) }},
				{Header: "Name", Extract: func(r any) string { return r.(netbox.Device).Name }},
				{Header: "Type", Extract: func(r any) string {
					d := r.(netbox.Device)
					if d.DeviceType == nil {
						return ""
					}
					mfr := ""
					if d.DeviceType.Manufacturer != nil {
						mfr = d.DeviceType.Manufacturer.Name + " "
					}
					return mfr + d.DeviceType.Model
				}},
				{Header: "Role", Extract: func(r any) string {
					if r.(netbox.Device).Role == nil {
						return ""
					}
					return r.(netbox.Device).Role.Name
				}},
				{Header: "Site", Extract: func(r any) string {
					if r.(netbox.Device).Site == nil {
						return ""
					}
					return r.(netbox.Device).Site.Name
				}},
				{Header: "Rack", Extract: func(r any) string {
					if r.(netbox.Device).Rack == nil {
						return ""
					}
					return r.(netbox.Device).Rack.Name
				}},
				{Header: "Status", Extract: func(r any) string { return r.(netbox.Device).Status.Label }},
				{Header: "Tenant", Extract: func(r any) string {
					if r.(netbox.Device).Tenant == nil {
						return ""
					}
					return r.(netbox.Device).Tenant.Name
				}},
			}
			return r.Render(io.Out, cols, rows)
		},
	}
}
