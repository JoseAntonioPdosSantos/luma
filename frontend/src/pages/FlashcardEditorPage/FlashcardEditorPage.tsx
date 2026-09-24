import { useEffect, useState, type ChangeEvent, type FormEvent } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import {
  createFlashcard,
  getFlashcard,
  updateFlashcard,
} from "../../services/flashcardService";
import { deleteAudio, uploadAudio } from "../../services/audioService";
import { apiUrl } from "../../services/api";
import { useApiErrorMessage } from "../../i18n/useApiErrorMessage";
import type { Audio } from "../../types";

const ACCEPTED_AUDIO_TYPES = ["audio/mpeg", "audio/wav", "audio/ogg"];

export function FlashcardEditorPage() {
  const { deckId, flashcardId } = useParams<{ deckId?: string; flashcardId?: string }>();
  const navigate = useNavigate();
  const { t } = useTranslation();
  const apiErrorMessage = useApiErrorMessage();
  const isEditing = Boolean(flashcardId);

  const [question, setQuestion] = useState("");
  const [answer, setAnswer] = useState("");
  const [hint, setHint] = useState("");
  const [exampleText, setExampleText] = useState("");
  const [exampleTranslation, setExampleTranslation] = useState("");
  const [ownerDeckId, setOwnerDeckId] = useState(deckId ?? "");
  const [audio, setAudio] = useState<Audio | null>(null);
  const [loading, setLoading] = useState(isEditing);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [audioError, setAudioError] = useState<string | null>(null);
  const [audioBusy, setAudioBusy] = useState(false);

  useEffect(() => {
    if (!flashcardId) return;
    getFlashcard(flashcardId)
      .then((card) => {
        setQuestion(card.question);
        setAnswer(card.answer);
        setHint(card.hint ?? "");
        setExampleText(card.extendedExample?.text ?? "");
        setExampleTranslation(card.extendedExample?.translation ?? "");
        setOwnerDeckId(card.deckId);
        setAudio(card.audio ?? null);
        setLoading(false);
      })
      .catch(() => {
        setError(t("flashcardEditor.loadError"));
        setLoading(false);
      });
  }, [flashcardId, t]);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);

    const input = {
      question,
      answer,
      hint,
      extendedExample:
        exampleText.trim() !== ""
          ? { text: exampleText, translation: exampleTranslation }
          : undefined,
    };

    try {
      if (isEditing && flashcardId) {
        await updateFlashcard(flashcardId, input);
      } else if (deckId) {
        await createFlashcard(deckId, input);
      }
      navigate(`/decks/${ownerDeckId}`);
    } catch (err) {
      setError(apiErrorMessage(err, "flashcardEditor.saveError"));
    } finally {
      setSubmitting(false);
    }
  }

  async function handleAudioChange(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    e.target.value = "";
    if (!file || !flashcardId) return;

    if (!ACCEPTED_AUDIO_TYPES.includes(file.type)) {
      setAudioError(t("flashcardEditor.audioTypeError"));
      return;
    }

    setAudioError(null);
    setAudioBusy(true);
    try {
      const updated = await uploadAudio(flashcardId, file);
      setAudio(updated.audio ?? null);
    } catch (err) {
      setAudioError(apiErrorMessage(err, "flashcardEditor.audioUploadError"));
    } finally {
      setAudioBusy(false);
    }
  }

  async function handleRemoveAudio() {
    if (!flashcardId) return;
    setAudioError(null);
    setAudioBusy(true);
    try {
      await deleteAudio(flashcardId);
      setAudio(null);
    } catch (err) {
      setAudioError(apiErrorMessage(err, "flashcardEditor.audioRemoveError"));
    } finally {
      setAudioBusy(false);
    }
  }

  const backTo = `/decks/${ownerDeckId || deckId}`;

  if (loading) {
    return (
      <main className="app-shell">
        <p className="loading">{t("flashcardEditor.loading")}</p>
      </main>
    );
  }

  return (
    <main className="app-shell">
      <Link to={backTo} className="back-link">
        &larr; {t("flashcardEditor.backToDeck")}
      </Link>
      <h1>{isEditing ? t("flashcardEditor.editTitle") : t("flashcardEditor.createTitle")}</h1>

      <form onSubmit={handleSubmit} className="form">
        <label htmlFor="question">{t("flashcardEditor.questionLabel")}</label>
        <textarea
          id="question"
          required
          maxLength={5000}
          value={question}
          onChange={(e) => setQuestion(e.target.value)}
          disabled={submitting}
        />

        <label htmlFor="answer">{t("flashcardEditor.answerLabel")}</label>
        <textarea
          id="answer"
          required
          maxLength={5000}
          value={answer}
          onChange={(e) => setAnswer(e.target.value)}
          disabled={submitting}
        />

        <label htmlFor="hint">{t("flashcardEditor.hintLabel")}</label>
        <textarea
          id="hint"
          maxLength={1000}
          value={hint}
          onChange={(e) => setHint(e.target.value)}
          disabled={submitting}
          placeholder={t("flashcardEditor.hintPlaceholder")}
        />

        <label htmlFor="example-text">{t("flashcardEditor.exampleLabel")}</label>
        <textarea
          id="example-text"
          maxLength={10000}
          value={exampleText}
          onChange={(e) => setExampleText(e.target.value)}
          disabled={submitting}
        />

        <label htmlFor="example-translation">{t("flashcardEditor.exampleTranslationLabel")}</label>
        <textarea
          id="example-translation"
          maxLength={10000}
          value={exampleTranslation}
          onChange={(e) => setExampleTranslation(e.target.value)}
          disabled={submitting}
        />

        {error && <p className="error" role="alert">{error}</p>}

        <div className="button-row">
          <button type="submit" disabled={submitting || question.trim() === "" || answer.trim() === ""}>
            {submitting ? t("flashcardEditor.saving") : t("flashcardEditor.save")}
          </button>
          <Link to={backTo}>
            <button type="button" className="secondary">
              {t("flashcardEditor.cancel")}
            </button>
          </Link>
        </div>
      </form>

      <div className="audio-section">
        <label htmlFor="audio-upload">{t("flashcardEditor.audioLabel")}</label>
        {!isEditing && (
          <p className="hint">{t("flashcardEditor.audioSaveFirst")}</p>
        )}
        {isEditing && (
          <>
            {audio && (
              <div className="button-row">
                <audio controls src={apiUrl(audio.url)} />
                <button type="button" className="secondary" disabled={audioBusy} onClick={handleRemoveAudio}>
                  {t("flashcardEditor.removeAudio")}
                </button>
              </div>
            )}
            <input
              id="audio-upload"
              type="file"
              accept={ACCEPTED_AUDIO_TYPES.join(",")}
              disabled={audioBusy}
              onChange={handleAudioChange}
            />
            {audioBusy && <p className="loading">{t("flashcardEditor.uploading")}</p>}
            {audioError && <p className="error" role="alert">{audioError}</p>}
          </>
        )}
      </div>
    </main>
  );
}
