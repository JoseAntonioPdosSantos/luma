// Soft, meaningful badge colors: every collection badge shares one calm
// tone and every group (folder) badge shares another, so color denotes
// what the badge represents (a collection vs. a group) rather than which
// specific one it is — see the Lumia design system's "Collection Cards"
// guidance against a different saturated hue per item.
export const DECK_BADGE_STYLE = { backgroundColor: "var(--color-primary-soft)", color: "var(--color-primary)" };
export const GROUP_BADGE_STYLE = { backgroundColor: "var(--color-lavender-soft)", color: "var(--color-lavender-fg)" };

export function initialsFor(name: string): string {
  const trimmed = name.trim();
  if (trimmed.length === 0) return "?";
  const words = trimmed.split(/\s+/);
  if (words.length === 1) return trimmed.slice(0, 2).toUpperCase();
  return (words[0][0] + words[1][0]).toUpperCase();
}
