package lists

func getLists() []List {
	return []List{
		{
			Name:        "dating:ut1-blacklists",
			Description: "Collection of websites blacklists managed by the Université Toulouse Capitole",
			URL:         "https://raw.githubusercontent.com/olbat/ut1-blacklists/refs/heads/master/blacklists/dating/domains",
			Category:    CategoryDating,
			Action:      "block",
		},
	}
}
