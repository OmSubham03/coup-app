package game

// VariantConfig holds UI info for a variant
type VariantConfig struct {
	Key              VariantKey          `json:"key"`
	Label            string              `json:"label"`
	Description      string              `json:"description"`
	Characters       []CharacterType     `json:"characters"`
	AvailableActions []ActionType        `json:"availableActions"`
	ActionGroups     ActionGroups        `json:"actionGroups"`
	ActionUI         map[ActionType]UIInfo `json:"actionUi"`
	CharacterRules   []CharacterRule     `json:"characterRules"`
}

type ActionGroups struct {
	Basic     []ActionType `json:"basic"`
	Character []ActionType `json:"character"`
	Targeted  []ActionType `json:"targeted"`
}

type UIInfo struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

type CharacterRule struct {
	Character CharacterType `json:"character"`
	Actions   []UIInfo      `json:"actions"`
	Blocks    []UIInfo      `json:"blocks"`
}

var actionUI = map[ActionType]UIInfo{
	ActionIncome:      {Label: "Income", Description: "+1 coin (safe)"},
	ActionForeignAid:  {Label: "Foreign Aid", Description: "+2 coins (blockable)"},
	ActionCoup:        {Label: "Coup", Description: "Pay 7 coins to kill influence (Unblockable)"},
	ActionTax:         {Label: "Tax (Duke)", Description: "+3 coins"},
	ActionAssassinate: {Label: "Assassinate (Assassin)", Description: "Pay 3 coins to kill influence"},
	ActionSteal:       {Label: "Steal (Captain)", Description: "Take 2 coins from opponent"},
	ActionExchange:    {Label: "Exchange (Ambassador)", Description: "Swap cards"},
	ActionInterrogate: {Label: "Interrogate (Inquisitor)", Description: "Reveal a card and optionally replace it"},
	ActionInquire:     {Label: "Inquire (Inquisitor)", Description: "Draw 1 card, return 1"},
}

func GetVariantConfig(variant VariantKey) VariantConfig {
	if variant == VariantInquisitor {
		return VariantConfig{
			Key:              VariantInquisitor,
			Label:            "Inquisitor",
			Description:      "Swap the Ambassador for the Inquisitor and interrogate opponents.",
			Characters:       []CharacterType{Duke, Assassin, Captain, Inquisitor, Contessa},
			AvailableActions: getAvailableActions(VariantInquisitor),
			ActionGroups: ActionGroups{
				Basic:     []ActionType{ActionIncome, ActionForeignAid},
				Character: []ActionType{ActionTax, ActionInquire},
				Targeted:  []ActionType{ActionInterrogate, ActionSteal, ActionAssassinate, ActionCoup},
			},
			ActionUI: actionUI,
			CharacterRules: []CharacterRule{
				{Character: Duke, Actions: []UIInfo{{Label: "Tax", Description: "Take 3 coins."}}, Blocks: []UIInfo{{Label: "Block Foreign Aid", Description: "Stops Foreign Aid."}}},
				{Character: Assassin, Actions: []UIInfo{{Label: "Assassinate", Description: "Pay 3 coins to kill an influence."}}, Blocks: []UIInfo{}},
				{Character: Captain, Actions: []UIInfo{{Label: "Steal", Description: "Take 2 coins from another player."}}, Blocks: []UIInfo{{Label: "Block Steal", Description: "Stops Steal."}}},
				{Character: Inquisitor, Actions: []UIInfo{{Label: "Inquire", Description: "Draw 1 card, then return 1 card."}, {Label: "Interrogate", Description: "Target reveals a card; you may swap it with the deck."}}, Blocks: []UIInfo{{Label: "Block Steal", Description: "Stops Steal."}}},
				{Character: Contessa, Actions: []UIInfo{}, Blocks: []UIInfo{{Label: "Block Assassination", Description: "Stops Assassination."}}},
			},
		}
	}

	return VariantConfig{
		Key:              VariantStandard,
		Label:            "Standard",
		Description:      "Classic Coup rules with the Ambassador.",
		Characters:       []CharacterType{Duke, Assassin, Captain, Ambassador, Contessa},
		AvailableActions: getAvailableActions(VariantStandard),
		ActionGroups: ActionGroups{
			Basic:     []ActionType{ActionIncome, ActionForeignAid},
			Character: []ActionType{ActionTax, ActionExchange},
			Targeted:  []ActionType{ActionSteal, ActionAssassinate, ActionCoup},
		},
		ActionUI: actionUI,
		CharacterRules: []CharacterRule{
			{Character: Duke, Actions: []UIInfo{{Label: "Tax", Description: "Take 3 coins."}}, Blocks: []UIInfo{{Label: "Block Foreign Aid", Description: "Stops Foreign Aid."}}},
			{Character: Assassin, Actions: []UIInfo{{Label: "Assassinate", Description: "Pay 3 coins to kill an influence."}}, Blocks: []UIInfo{}},
			{Character: Captain, Actions: []UIInfo{{Label: "Steal", Description: "Take 2 coins from another player."}}, Blocks: []UIInfo{{Label: "Block Steal", Description: "Stops Steal."}}},
			{Character: Ambassador, Actions: []UIInfo{{Label: "Exchange", Description: "Draw 2 cards, return 2."}}, Blocks: []UIInfo{{Label: "Block Steal", Description: "Stops Steal."}}},
			{Character: Contessa, Actions: []UIInfo{}, Blocks: []UIInfo{{Label: "Block Assassination", Description: "Stops Assassination."}}},
		},
	}
}
