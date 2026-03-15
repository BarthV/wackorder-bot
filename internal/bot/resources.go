package bot

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

// resources is the static list of orderable components.
var resources = []string{
	"Agricium",
	"Aluminum",
	"Aphorite",
	"Beradom",
	"Beryl",
	"Bexalite",
	"Borase",
	"Cobalt",
	"Copper",
	"Corundum",
	"Diamond",
	"Dolivine",
	"Feynmaline",
	"Glacosite",
	"Gold",
	"Hadanite",
	"Hephaestanite",
	"Iron",
	"Jaclium",
	"Laranite",
	"Lindinium",
	"Quantainium",
	"Quartz",
	"Raw Ice",
	"Riccite",
	"Savrilium",
	"Silicon",
	"Stileron",
	"Taranite",
	"Tin",
	"Titanium",
	"Torite",
	"Tungsten",
}

// handleOrderAutocomplete responds to autocomplete requests for the /order component option.
func (h *handler) handleOrderAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var query string
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == "component" && opt.Focused {
			query = opt.StringValue()
			break
		}
	}

	matches := filterResources(query)
	choices := make([]*discordgo.ApplicationCommandOptionChoice, len(matches))
	for j, name := range matches {
		choices[j] = &discordgo.ApplicationCommandOptionChoice{Name: name, Value: name}
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{ //nolint:errcheck
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{Choices: choices},
	})
}

// isValidResource reports whether name exactly matches a resource (case-insensitive).
func isValidResource(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, r := range resources {
		if strings.ToLower(r) == name {
			return true
		}
	}
	return false
}

// filterResources returns up to 25 resources whose name contains the query (case-insensitive).
func filterResources(query string) []string {
	query = strings.ToLower(strings.TrimSpace(query))
	var out []string
	for _, r := range resources {
		if query == "" || strings.Contains(strings.ToLower(r), query) {
			out = append(out, r)
			if len(out) == 25 {
				break
			}
		}
	}
	return out
}
