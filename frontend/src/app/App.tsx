import { BrowserRouter, Route, Routes } from "react-router-dom";
import { AuthProvider } from "../hooks/useAuth";
import { RequireAuth } from "../components/RequireAuth";
import { LoginPage } from "../pages/LoginPage/LoginPage";
import { DashboardPage } from "../pages/DashboardPage/DashboardPage";
import { DeckPage } from "../pages/DeckPage/DeckPage";
import { GroupPage } from "../pages/GroupPage/GroupPage";
import { FlashcardEditorPage } from "../pages/FlashcardEditorPage/FlashcardEditorPage";
import { StudyPage } from "../pages/StudyPage/StudyPage";
import { StatsPage } from "../pages/StatsPage/StatsPage";
import { SettingsPage } from "../pages/SettingsPage/SettingsPage";

export function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          <Route path="/" element={<LoginPage />} />
          <Route
            path="/dashboard"
            element={
              <RequireAuth>
                <DashboardPage />
              </RequireAuth>
            }
          />
          <Route
            path="/settings"
            element={
              <RequireAuth>
                <SettingsPage />
              </RequireAuth>
            }
          />
          <Route
            path="/decks/:deckId"
            element={
              <RequireAuth>
                <DeckPage />
              </RequireAuth>
            }
          />
          <Route
            path="/groups/:groupId"
            element={
              <RequireAuth>
                <GroupPage />
              </RequireAuth>
            }
          />
          <Route
            path="/decks/:deckId/study"
            element={
              <RequireAuth>
                <StudyPage />
              </RequireAuth>
            }
          />
          <Route
            path="/decks/:deckId/stats"
            element={
              <RequireAuth>
                <StatsPage />
              </RequireAuth>
            }
          />
          <Route
            path="/decks/:deckId/flashcards/new"
            element={
              <RequireAuth>
                <FlashcardEditorPage />
              </RequireAuth>
            }
          />
          <Route
            path="/flashcards/:flashcardId/edit"
            element={
              <RequireAuth>
                <FlashcardEditorPage />
              </RequireAuth>
            }
          />
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  );
}
