// Longest answer (in characters) an automatic hint will spell out; anything
// beyond is cut and marked with an ellipsis so a long answer stays a hint
// rather than a wall of underscores.
const MAX_LENGTH = 120;

const isLetterOrDigit = (ch: string) => /[\p{L}\p{N}]/u.test(ch);

// Masks one word: the first letter stays, every other letter/digit becomes
// "_", punctuation is kept. Characters are separated by a space so the
// underscores can be counted.
function maskWord(word: string): string {
  let seenLetter = false;
  return Array.from(word)
    .map((ch) => {
      if (!isLetterOrDigit(ch)) return ch;
      if (!seenLetter) {
        seenLetter = true;
        return ch;
      }
      return "_";
    })
    .join(" ");
}

// Builds a hint for a card that has none of its own: the first letter of
// each word of the answer and one underscore per remaining letter, e.g.
// "rescue" -> "r _ _ _ _ _". Words are separated by three spaces.
export function generateAutoHint(answer: string): string {
  const trimmed = answer.trim();
  if (trimmed === "") return "";

  const truncated = trimmed.length > MAX_LENGTH;
  const words = trimmed.slice(0, MAX_LENGTH).split(/\s+/).map(maskWord);
  const hint = words.join("   ");
  return truncated ? `${hint} …` : hint;
}
