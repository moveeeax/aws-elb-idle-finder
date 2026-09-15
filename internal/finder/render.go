package finder

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
)

// WriteJSON renders the report as indented JSON.
func (r Report) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// WriteTable renders the report as an aligned human-readable table with a
// summary footer.
func (r Report) WriteTable(w io.Writer) error {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "STATUS\tKIND\tNAME\tHEALTHY\tTRAFFIC\tWASTE/MO\tREASON")
	for _, f := range r.Findings {
		traffic := fmt.Sprintf("%.0f", f.Traffic)
		if f.Traffic < 0 {
			traffic = "-"
		}
		waste := "-"
		if f.MonthlyWasteUSD > 0 {
			waste = fmt.Sprintf("$%.2f", f.MonthlyWasteUSD)
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%s\t%s\t%s\n",
			f.Status, f.Kind, f.Name, f.HealthyTargets, traffic, waste, f.Reason)
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	fmt.Fprintf(w, "\n%s (%s lookback): %d idle load balancer(s), ~$%.2f/mo wasted.\n",
		r.Region, r.Lookback, r.IdleCount, r.TotalWasteUSD)
	return nil
}
