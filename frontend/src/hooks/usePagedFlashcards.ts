import { useCallback, useEffect, useRef, useState } from "react";
import { listFlashcards } from "../services/flashcardService";
import type { Flashcard } from "../types";

export const PAGE_SIZE = 30;

type Status = "idle" | "loading" | "loaded" | "error";

interface Options {
  deckId: string | undefined;
  archived?: boolean;
  query?: string;
  // When false nothing is fetched (e.g. a collapsed "archived" section).
  enabled?: boolean;
}

// Loads a deck's flashcards one page at a time. The first page is fetched
// whenever the deck, the archived flag or the search text changes; loadMore
// appends the next page. Responses that arrive after a newer request was
// started (a fast typist, a slow network) are discarded.
export function usePagedFlashcards({ deckId, archived = false, query = "", enabled = true }: Options) {
  const [items, setItems] = useState<Flashcard[]>([]);
  const [total, setTotal] = useState(0);
  const [status, setStatus] = useState<Status>("idle");
  const [loadingMore, setLoadingMore] = useState(false);
  const [loadMoreFailed, setLoadMoreFailed] = useState(false);
  const [reloadTick, setReloadTick] = useState(0);
  const latestRequest = useRef(0);
  const loadedFor = useRef("");

  useEffect(() => {
    if (!deckId || !enabled) return;

    // A different deck (or list) must not show the previous one's cards
    // while its own first page is loading; a new search text may keep the
    // old rows until the results arrive.
    const scope = `${deckId}|${archived}`;
    if (loadedFor.current !== scope) {
      loadedFor.current = scope;
      setItems([]);
      setTotal(0);
    }

    const request = ++latestRequest.current;
    setStatus("loading");
    setLoadMoreFailed(false);
    listFlashcards(deckId, { archived, query, limit: PAGE_SIZE, offset: 0 })
      .then((page) => {
        if (request !== latestRequest.current) return;
        setItems(page.items);
        setTotal(page.total);
        setStatus("loaded");
      })
      .catch(() => {
        if (request === latestRequest.current) setStatus("error");
      });
  }, [deckId, archived, query, enabled, reloadTick]);

  const loadMore = useCallback(async () => {
    if (!deckId || loadingMore || items.length >= total) return;
    const request = latestRequest.current;
    setLoadingMore(true);
    setLoadMoreFailed(false);
    try {
      const page = await listFlashcards(deckId, { archived, query, limit: PAGE_SIZE, offset: items.length });
      if (request !== latestRequest.current) return;
      setItems((current) => {
        const seen = new Set(current.map((c) => c.id));
        return [...current, ...page.items.filter((c) => !seen.has(c.id))];
      });
      setTotal(page.total);
    } catch {
      if (request === latestRequest.current) setLoadMoreFailed(true);
    } finally {
      setLoadingMore(false);
    }
  }, [deckId, archived, query, items.length, total, loadingMore]);

  // Refetches the first page (e.g. after a card was restored).
  const reload = useCallback(() => setReloadTick((tick) => tick + 1), []);

  // Drops a card that was just archived/restored/deleted without waiting
  // for a refetch.
  const removeLocally = useCallback((id: string) => {
    setItems((current) => current.filter((c) => c.id !== id));
    setTotal((current) => Math.max(0, current - 1));
  }, []);

  return { items, total, status, loadingMore, loadMoreFailed, loadMore, reload, removeLocally };
}
