package mcp

import (
	"fmt"
	"strings"

	"github.com/drewjocham/mongo-migration-tool/internal/migration"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func formatStatusTable(status []migration.MigrationStatus) string {
	var b strings.Builder
	b.WriteString("### Migration Status\n\n")
	b.WriteString("| Version | Status | Applied At | Description |\n")
	b.WriteString("| :--- | :--- | :--- | :--- |\n")

	for _, st := range status {
		applied := "⏳ Pending"
		appliedAt := "N/A"

		if st.Applied {
			applied = "✅ Applied"
			if st.AppliedAt != nil {
				appliedAt = st.AppliedAt.Format("2006-01-02 15:04")
			}
		}

		b.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
			st.Version, applied, appliedAt, st.Description))
	}
	return b.String()
}

func formatIndexKeys(keys interface{}) string {
	var keyParts []string
	if doc, ok := keys.(bson.D); ok {
		for _, elem := range doc {
			keyParts = append(keyParts, fmt.Sprintf("%s: %v", elem.Key, elem.Value))
		}
	}
	if m, ok := keys.(bson.M); ok {
		for k, v := range m {
			keyParts = append(keyParts, fmt.Sprintf("%s: %v", k, v))
		}
	}

	if len(keyParts) == 0 {
		return "none"
	}
	return strings.Join(keyParts, ", ")
}
