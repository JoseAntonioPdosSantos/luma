import { useState } from "react";
import { useTranslation } from "react-i18next";
import { RestoreIcon, TrashIcon } from "./Icons";

interface ArchivedActionsProps {
  restoreLabel: string;
  deleteLabel: string;
  // Shown while asking the user to confirm the permanent deletion.
  confirmMessage: string;
  onRestore: () => void;
  onDelete: () => void;
}

// Restore / permanently-delete buttons for one archived item. Deleting is
// irreversible, so it takes a second, explicit confirmation in the row
// itself (no browser dialog, which is awkward on mobile).
export function ArchivedActions({
  restoreLabel,
  deleteLabel,
  confirmMessage,
  onRestore,
  onDelete,
}: ArchivedActionsProps) {
  const [confirming, setConfirming] = useState(false);
  const { t } = useTranslation();

  if (confirming) {
    return (
      <div className="archived-actions archived-confirm" role="group" aria-label={t("archivedActions.confirmAria")}>
        <p className="archived-confirm-message">{confirmMessage}</p>
        <div className="archived-actions-buttons">
          <button className="secondary" onClick={() => setConfirming(false)}>
            {t("common.cancel")}
          </button>
          <button className="danger" onClick={onDelete}>
            {t("archivedActions.deletePermanently")}
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="archived-actions">
      <div className="archived-actions-buttons">
        <button className="secondary action-restore" onClick={onRestore}>
          <RestoreIcon />
          {restoreLabel}
        </button>
        <button className="secondary action-archive" onClick={() => setConfirming(true)}>
          <TrashIcon />
          {deleteLabel}
        </button>
      </div>
    </div>
  );
}
