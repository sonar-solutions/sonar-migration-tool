// Temporary diagnostic: decode real scanner-report .pb files using the repo's
// own proto bindings. Delete after use.
package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"sort"

	pb "github.com/sonar-solutions/sonar-migration-tool/internal/scanreport/proto"
	"google.golang.org/protobuf/proto"
)

func readDelimited(b []byte) [][]byte {
	var out [][]byte
	for len(b) > 0 {
		n, adv := binary.Uvarint(b)
		if adv <= 0 {
			break
		}
		b = b[adv:]
		if uint64(len(b)) < n {
			break
		}
		out = append(out, b[:n])
		b = b[n:]
	}
	return out
}

func truncate(s string) string {
	if len(s) > 50 {
		return fmt.Sprintf("%s...(%d chars)", s[:50], len(s))
	}
	return s
}

func measVal(m *pb.Measure) string {
	if v := m.GetIntValue(); v != nil {
		return fmt.Sprintf("int=%d data=%q", v.GetValue(), truncate(v.GetData()))
	}
	if v := m.GetLongValue(); v != nil {
		return fmt.Sprintf("long=%d data=%q", v.GetValue(), truncate(v.GetData()))
	}
	if v := m.GetDoubleValue(); v != nil {
		return fmt.Sprintf("double=%v data=%q", v.GetValue(), truncate(v.GetData()))
	}
	if v := m.GetStringValue(); v != nil {
		return fmt.Sprintf("string=%q", truncate(v.GetValue()))
	}
	if v := m.GetBooleanValue(); v != nil {
		return fmt.Sprintf("bool=%v data=%q", v.GetValue(), truncate(v.GetData()))
	}
	return "(empty)"
}

func main() {
	mode := os.Args[1]
	path := os.Args[2]
	raw, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	switch mode {
	case "measures":
		msgs := readDelimited(raw)
		fmt.Printf("%d measure messages in %s\n", len(msgs), path)
		keys := map[string]string{}
		for _, m := range msgs {
			var meas pb.Measure
			if err := proto.Unmarshal(m, &meas); err != nil {
				continue
			}
			keys[meas.GetMetricKey()] = measVal(&meas)
		}
		var order []string
		for k := range keys {
			order = append(order, k)
		}
		sort.Strings(order)
		for _, k := range order {
			fmt.Printf("  %-28s %s\n", k, keys[k])
		}
	case "component":
		var c pb.Component
		if err := proto.Unmarshal(raw, &c); err != nil {
			panic(err)
		}
		fmt.Printf("ref=%d type=%s key=%q name=%q lines=%d lang=%q nChildren=%d childRefs=%v\n",
			c.GetRef(), c.GetType(), c.GetKey(), c.GetName(), c.GetLines(), c.GetLanguage(), len(c.GetChildRef()), c.GetChildRef())
	default:
		fmt.Println("unknown mode")
	}
}
