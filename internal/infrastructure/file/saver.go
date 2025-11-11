package file

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/alexperezortuno/cloudrip/internal/domain"
)

func SaveText(path string, m map[string][]domain.ResultEntry) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// salida ordenada por fqdn
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	w := bufio.NewWriter(f)
	for _, k := range keys {
		entries := m[k]
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].Type == entries[j].Type {
				return entries[i].IP < entries[j].IP
			}
			return entries[i].Type < entries[j].Type
		})
		for _, e := range entries {
			fmt.Fprintf(w, "%s -> %s (%s)\n", e.FQDN, e.IP, e.Type)
		}
	}
	return w.Flush()
}

func SaveJSON(path string, m map[string][]domain.ResultEntry) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var flat []domain.ResultEntry
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		entries := m[k]
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].Type == entries[j].Type {
				return entries[i].IP < entries[j].IP
			}
			return entries[i].Type < entries[j].Type
		})
		flat = append(flat, entries...)
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(flat)
}
