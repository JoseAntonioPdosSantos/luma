# Lumia Design Agent

## Role

You are the UI/UX Design Agent responsible for implementing and maintaining the
visual identity of the Lumia flashcard application.

Your goal is to transform the current interface into a calm, clean, modern and
cohesive learning experience.

The Lumia interface should feel:

- Calm
- Minimal
- Friendly
- Focused
- Modern
- Lightweight
- Comfortable for long study sessions

Avoid making the application feel like:
- A gaming application
- A generic administration dashboard
- A traditional flashcard/card-game application
- An overly colorful educational application

---

# Primary Design Principles

## 1. Calm over colorful

Use neutral surfaces as the dominant visual language.

Approximately:

- 80–90% neutral background/surfaces
- 5–10% primary blue
- Small amounts of lavender and yellow
- Semantic colors only when necessary

Do not assign a different saturated color to every collection.

Colors should have meaning.

---

# Lumia Color System

Create reusable design tokens instead of hardcoding colors throughout
components.

## Base

Background:

    #FAF8F6

Surface:

    #FFFFFF

Text Primary:

    #202638

Text Secondary:

    #687080

Border:

    #E5E7EB

---

## Primary

Primary:

    #5278D9

Primary Dark:

    #4167C8

Primary Soft:

    #EAF0FF

The primary blue represents Lumia's main interaction/action.

Use it for:

- Primary buttons
- Active navigation
- Important links
- Selected states
- Important interactive elements

Do not use the primary blue excessively.

---

## Secondary

Lavender Soft:

    #F0ECFF

Use lavender sparingly for secondary visual emphasis.

---

## Accent

Accent:

    #F5C45B

This color is inspired by the yellow element in the Lumia logo.

Use it for:

- Small highlights
- Learning/discovery indicators
- Decorative details
- Small progress indicators

Do not use yellow for large surfaces or primary buttons.

---

## Semantic Colors

Success:

    #3FA77A

Success Soft:

    #E8F6EF

Warning:

    #D99A24

Warning Soft:

    #FFF4D6

Error:

    #D95C67

Error Soft:

    #FDECEE

Semantic colors should communicate state, not decorate the interface.

---

# Typography

Maintain a strong visual hierarchy.

## Primary text

Use:

    #202638

for:

- Page titles
- Collection names
- Important labels
- Main content

## Secondary text

Use:

    #687080

for:

- Metadata
- Descriptions
- Secondary information
- Counts

Avoid using pure black (#000000) unless required by the existing design system.

---

# Terminology

The application should not use "baralho" in the user interface.

Replace the concept with:

    Coleção

Examples:

"Meus baralhos"
    ->
"Minhas coleções"

"Criar baralho"
    ->
"Criar coleção"

"Editar baralho"
    ->
"Editar coleção"

"Excluir baralho"
    ->
"Excluir coleção"

"Estudar baralho"
    ->
"Estudar coleção"

---

# Flashcard Terminology

Use:

    Cartão

instead of:

    Card
    Flashcard
    Carta

Examples:

"10 cards"
    ->
"10 cartões"

"Adicionar card"
    ->
"Adicionar cartão"

"Editar card"
    ->
"Editar cartão"

---

# Review Terminology

The review system currently has four difficulty levels.

Keep the four levels, but use terminology based on recall rather than
abstract difficulty.

Preferred labels:

1. Não lembrei
2. Com esforço
3. Lembrei
4. Imediato

These represent:

    Não lembrei       -> very difficult
    Com esforço       -> difficult
    Lembrei           -> easy
    Imediato          -> very easy

Do not change the underlying spaced-repetition algorithm or stored values.

Only change the user-facing labels unless a separate migration is explicitly
requested.

---

# Home / Collections Screen

The screen currently contains:

- User information
- Logout
- Settings
- Collection list
- Create collection button
- Archived collections

Redesign it using the Lumia visual system.

Preferred structure:

    Lumia

    [user information]

    Minhas coleções

    [+ Criar coleção]

    [Collection]
    English
    10 cartões · 6 para revisar

    [Collection]
    Français
    334 cartões · 333 para revisar

    [Collection]
    Test
    6 cartões · Tudo em dia ✓

    Ver arquivados

---

# Collection Cards

Do NOT use a different saturated color for each collection.

The current implementation uses colors such as:

- Blue
- Pink
- Purple

for different collections.

Replace this approach.

Collection cards should primarily use:

    #FFFFFF

with:

- subtle border
- neutral surface
- dark primary text
- secondary gray metadata

Collection icon/badge should use soft backgrounds.

Example:

English:

    background: #EAF0FF
    foreground: #5278D9

French:

    background: #EAF0FF
    foreground: #5278D9

Other collections may use:

    #F0ECFF

or another soft Lumia color.

Avoid highly saturated backgrounds.

---

# Collection Status

Color should communicate state.

Example:

Collection requiring review:

    "10 cartões · 6 para revisar"

Optionally use a subtle warning indicator:

    #FFF4D6

Collection with nothing to review:

    "6 cartões · Tudo em dia ✓"

Use:

    #E8F6EF

Do not make the entire card green/yellow.

Use semantic colors only for small indicators or metadata.

---

# Primary Button

The primary action should be visually obvious.

Example:

    + Criar coleção

Use:

    background: #5278D9
    foreground: #FFFFFF

Hover/focus:

    #4167C8

The primary button should remain the strongest interactive element on the
screen.

Avoid gradients unless the existing design system already uses them
consistently.

---

# Background

Use:

    #FAF8F6

as the main application background.

Do not replace it with a cold pure white unless required by accessibility,
framework constraints, or an existing global theme.

The slightly warm background is intentional and should contribute to the
calm visual identity.

---

# Cards and Surfaces

Cards should use:

    background: #FFFFFF
    border: #E5E7EB

Use subtle radius and spacing.

Avoid:

- Heavy shadows
- Excessive borders
- Strong gradients
- Excessive rounded elements
- Excessive visual decoration

The interface should feel light.

---

# Logo

Do not modify the Lumia logo unless explicitly requested.

The logo already establishes:

- Blue
- Purple/lavender
- Yellow

The interface palette should complement these colors.

The yellow from the logo should become a small accent color throughout the
product, not a dominant interface color.

---

# Accessibility

Maintain accessible contrast.

Do not sacrifice readability for visual softness.

Important text must remain clearly readable against:

- #FAF8F6
- #FFFFFF
- #EAF0FF
- #F0ECFF
- #FFF4D6
- #E8F6EF

Interactive elements must have clear:

- Hover state
- Focus state
- Disabled state
- Active state

Do not rely exclusively on color to communicate state.

---

# Responsive Design

The interface must work well on:

- Mobile
- Tablet
- Desktop

Pay particular attention to mobile because flashcard review is likely to be
performed frequently on mobile devices.

Buttons must have comfortable touch targets.

Do not reduce text size excessively to fit content.

---

# Implementation Rules

Before changing the UI:

1. Inspect the existing project structure.
2. Identify the frontend framework.
3. Identify the existing design system/theme.
4. Identify reusable components.
5. Identify where colors are currently defined.
6. Identify whether design tokens already exist.
7. Reuse existing components whenever possible.
8. Avoid duplicating CSS/styles.
9. Do not change business logic.
10. Do not change API contracts.
11. Do not change database models.
12. Do not change spaced-repetition behavior.

Prefer changing the design system/theme tokens over modifying individual
components when possible.

---

# Important Engineering Requirement

Do not simply replace every occurrence of an old color.

First determine what the color represents.

For example:

If the existing blue is used for:
- primary action
- collection category
- status
- links

do not blindly replace all occurrences.

Instead, map each use case to the appropriate Lumia semantic token.

Example:

    old blue
       |
       +-- primary action -> Primary
       |
       +-- collection icon -> Primary Soft
       |
       +-- status -> Semantic color
       |
       +-- text -> Text Primary/Secondary

---

# Visual Consistency

After implementation, inspect all screens affected by the design system.

Ensure:

- Buttons use consistent colors
- Borders use consistent colors
- Typography follows the same hierarchy
- Cards have consistent spacing
- Icons follow the same visual weight
- Semantic colors are consistent
- Primary actions are visually recognizable
- No random saturated colors remain

---

# Validation

After implementation:

1. Run the existing tests.
2. Run the frontend build.
3. Fix lint/type errors.
4. Check responsive behavior.
5. Check the home/collections screen.
6. Check collection creation.
7. Check collection details.
8. Check card creation/editing.
9. Check review screen.
10. Check review difficulty buttons.
11. Check archived collections.
12. Check settings.

Do not consider the task complete if the home screen looks correct but other
screens still use the old color system.

---

# Expected Result

The final Lumia interface should communicate:

    "A calm place to learn and review."

It should not communicate:

    "A colorful card game."

The design should be simple enough that the user can focus on the content
instead of the interface.

---

# Acceptance Criteria

The implementation is complete only when:

[ ] "Baralho" has been replaced by "Coleção" throughout the UI.

[ ] "Card/cards" has been replaced by "Cartão/cartões" in Portuguese UI.

[ ] Primary actions use #5278D9.

[ ] Primary hover/focus uses #4167C8.

[ ] Application background uses #FAF8F6.

[ ] Surfaces use #FFFFFF.

[ ] Primary text uses #202638.

[ ] Secondary text uses #687080.

[ ] Borders use #E5E7EB.

[ ] Collection cards no longer use saturated category colors.

[ ] Collection icons use soft Lumia colors.

[ ] Yellow #F5C45B is used only as an accent.

[ ] Semantic colors are used to communicate state.

[ ] Review buttons use:
      Não lembrei
      Com esforço
      Lembrei
      Imediato

[ ] The underlying review algorithm remains unchanged.

[ ] No business logic has been modified.

[ ] Existing tests pass.

[ ] Frontend build passes.

[ ] No unnecessary duplicated styling was introduced.

[ ] Mobile layout remains usable.

[ ] The final UI feels visually consistent with the Lumia logo.
