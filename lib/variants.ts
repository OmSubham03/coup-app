import type { ActionType, CharacterType } from "./game-logic";

export type VariantKey = "standard" | "inquisitor";

export interface VariantActionRequirement {
    character?: CharacterType;
    cost?: number;
    needsTarget?: boolean;
    canBeBlocked?: boolean;
    blockingCharacters?: CharacterType[];
}

export interface ActionUiInfo {
    label: string;
    description: string;
}

export interface CharacterRuleEntry {
    character: CharacterType;
    actions: ActionUiInfo[];
    blocks: ActionUiInfo[];
}

export interface VariantConfig {
    key: VariantKey;
    label: string;
    description: string;
    characters: CharacterType[];
    availableActions: ActionType[];
    actionGroups: {
        basic: ActionType[];
        character: ActionType[];
        targeted: ActionType[];
    };
    actionRequirements: Record<ActionType, VariantActionRequirement>;
    actionUi: Record<ActionType, ActionUiInfo>;
    generalActions: ActionUiInfo[];
    characterRules: CharacterRuleEntry[];
}

export const CHARACTER_IMAGES: Record<CharacterType, string> = {
    Duke: "/textures/duke.jpg",
    Assassin: "/textures/assassin.jpg",
    Captain: "/textures/captain.jpg",
    Ambassador: "/textures/ambassador.jpg",
    Contessa: "/textures/contessa.jpg",
    Inquisitor: "/textures/inquisitor.png",
};

const ACTION_UI: Record<ActionType, ActionUiInfo> = {
    income: { label: "Income", description: "+1 coin (safe)" },
    foreign_aid: { label: "Foreign Aid", description: "+2 coins (blockable)" },
    coup: { label: "Coup", description: "Pay 7 coins to kill influence (Unblockable)" },
    tax: { label: "Tax (Duke)", description: "+3 coins" },
    assassinate: { label: "Assassinate (Assassin)", description: "Pay 3 coins to kill influence" },
    steal: { label: "Steal (Captain)", description: "Take 2 coins from opponent" },
    exchange: { label: "Exchange (Ambassador)", description: "Swap cards" },
    interrogate: { label: "Interrogate (Inquisitor)", description: "Reveal a card and optionally replace it" },
    inquire: { label: "Inquire (Inquisitor)", description: "Draw 1 card, return 1" },
};

const GENERAL_ACTIONS: ActionUiInfo[] = [
    { label: "Income", description: "Take 1 coin. Cannot be blocked." },
    { label: "Foreign Aid", description: "Take 2 coins. Can be blocked by Duke." },
    { label: "Coup", description: "Pay 7 coins. Choose a player to lose influence. Unblockable." },
];

const STANDARD_ACTION_REQUIREMENTS: Record<ActionType, VariantActionRequirement> = {
    income: {},
    foreign_aid: {
        canBeBlocked: true,
        blockingCharacters: ["Duke"],
    },
    coup: {
        cost: 7,
        needsTarget: true,
    },
    tax: {
        character: "Duke",
    },
    assassinate: {
        character: "Assassin",
        cost: 3,
        needsTarget: true,
        canBeBlocked: true,
        blockingCharacters: ["Contessa"],
    },
    steal: {
        character: "Captain",
        needsTarget: true,
        canBeBlocked: true,
        blockingCharacters: ["Captain", "Ambassador"],
    },
    exchange: {
        character: "Ambassador",
    },
    interrogate: {
        character: "Inquisitor",
        needsTarget: true,
    },
    inquire: {
        character: "Inquisitor",
    },
};

const INQUISITOR_ACTION_REQUIREMENTS: Record<ActionType, VariantActionRequirement> = {
    ...STANDARD_ACTION_REQUIREMENTS,
    steal: {
        character: "Captain",
        needsTarget: true,
        canBeBlocked: true,
        blockingCharacters: ["Captain", "Inquisitor"],
    },
    interrogate: {
        character: "Inquisitor",
        needsTarget: true,
    },
    inquire: {
        character: "Inquisitor",
    },
};

const STANDARD_AVAILABLE_ACTIONS: ActionType[] = [
    "income",
    "foreign_aid",
    "coup",
    "tax",
    "assassinate",
    "steal",
    "exchange",
];

const INQUISITOR_AVAILABLE_ACTIONS: ActionType[] = [
    "income",
    "foreign_aid",
    "coup",
    "tax",
    "assassinate",
    "steal",
    "interrogate",
    "inquire",
];

export const VARIANT_CONFIGS: Record<VariantKey, VariantConfig> = {
    standard: {
        key: "standard",
        label: "Standard",
        description: "Classic Coup rules with the Ambassador.",
        characters: ["Duke", "Assassin", "Captain", "Ambassador", "Contessa"],
        availableActions: STANDARD_AVAILABLE_ACTIONS,
        actionGroups: {
            basic: ["income", "foreign_aid"],
            character: ["tax", "exchange"],
            targeted: ["steal", "assassinate", "coup"],
        },
        actionRequirements: STANDARD_ACTION_REQUIREMENTS,
        actionUi: ACTION_UI,
        generalActions: GENERAL_ACTIONS,
        characterRules: [
            {
                character: "Duke",
                actions: [{ label: "Tax", description: "Take 3 coins." }],
                blocks: [{ label: "Block Foreign Aid", description: "Stops Foreign Aid." }],
            },
            {
                character: "Assassin",
                actions: [{ label: "Assassinate", description: "Pay 3 coins to kill an influence." }],
                blocks: [],
            },
            {
                character: "Captain",
                actions: [{ label: "Steal", description: "Take 2 coins from another player." }],
                blocks: [{ label: "Block Steal", description: "Stops Steal." }],
            },
            {
                character: "Ambassador",
                actions: [{ label: "Exchange", description: "Draw 2 cards, return 2." }],
                blocks: [{ label: "Block Steal", description: "Stops Steal." }],
            },
            {
                character: "Contessa",
                actions: [],
                blocks: [{ label: "Block Assassination", description: "Stops Assassination." }],
            },
        ],
    },
    inquisitor: {
        key: "inquisitor",
        label: "Inquisitor",
        description: "Swap the Ambassador for the Inquisitor and interrogate opponents.",
        characters: ["Duke", "Assassin", "Captain", "Inquisitor", "Contessa"],
        availableActions: INQUISITOR_AVAILABLE_ACTIONS,
        actionGroups: {
            basic: ["income", "foreign_aid"],
            character: ["tax", "inquire"],
            targeted: ["interrogate", "steal", "assassinate", "coup"],
        },
        actionRequirements: INQUISITOR_ACTION_REQUIREMENTS,
        actionUi: ACTION_UI,
        generalActions: GENERAL_ACTIONS,
        characterRules: [
            {
                character: "Duke",
                actions: [{ label: "Tax", description: "Take 3 coins." }],
                blocks: [{ label: "Block Foreign Aid", description: "Stops Foreign Aid." }],
            },
            {
                character: "Assassin",
                actions: [{ label: "Assassinate", description: "Pay 3 coins to kill an influence." }],
                blocks: [],
            },
            {
                character: "Captain",
                actions: [{ label: "Steal", description: "Take 2 coins from another player." }],
                blocks: [{ label: "Block Steal", description: "Stops Steal." }],
            },
            {
                character: "Inquisitor",
                actions: [
                    { label: "Inquire", description: "Draw 1 card, then return 1 card." },
                    { label: "Interrogate", description: "Target reveals a card; you may swap it with the deck." },
                ],
                blocks: [{ label: "Block Steal", description: "Stops Steal." }],
            },
            {
                character: "Contessa",
                actions: [],
                blocks: [{ label: "Block Assassination", description: "Stops Assassination." }],
            },
        ],
    },
};

export function isVariantKey(value?: string | null): value is VariantKey {
    return value === "standard" || value === "inquisitor";
}

export function normalizeVariant(value?: string | null): VariantKey {
    return isVariantKey(value) ? value : "standard";
}

export function getVariantConfig(value?: string | null): VariantConfig {
    return VARIANT_CONFIGS[normalizeVariant(value)];
}
