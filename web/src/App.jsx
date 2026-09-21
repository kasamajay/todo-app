import { useEffect, useState } from 'react'
import { api, getToken, clearToken, onUnauthorized } from './api.js'
import { colors, fontFamily } from './theme.js'
import Login from './components/Login.jsx'
import BoardsList from './components/BoardsList.jsx'
import Kanban from './components/Kanban.jsx'

export default function App() {
  const [view, setView] = useState('loading') // loading | login | boards | kanban
  const [user, setUser] = useState(null)
  const [activeBoard, setActiveBoard] = useState(null)

  useEffect(() => {
    onUnauthorized(() => {
      setUser(null)
      setActiveBoard(null)
      setView('login')
    })

    if (!getToken()) {
      setView('login')
      return
    }
    api
      .me()
      .then((u) => {
        setUser(u)
        setView('boards')
      })
      .catch(() => {
        clearToken()
        setView('login')
      })
  }, [])

  function handleLoginSuccess(loggedInUser) {
    setUser(loggedInUser)
    setView('boards')
  }

  function handleSelectBoard(board) {
    setActiveBoard(board)
    setView('kanban')
  }

  function handleBackToBoards() {
    setActiveBoard(null)
    setView('boards')
  }

  async function handleLogout() {
    try {
      await api.logout()
    } catch {
      // ignore - we're clearing the token client-side regardless
    }
    clearToken()
    setUser(null)
    setActiveBoard(null)
    setView('login')
  }

  if (view === 'loading') {
    return (
      <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', fontFamily, color: colors.textMuted }}>
        Loading…
      </div>
    )
  }

  if (view === 'login') {
    return <Login onSuccess={handleLoginSuccess} />
  }

  if (view === 'boards') {
    return <BoardsList user={user} onSelectBoard={handleSelectBoard} onLogout={handleLogout} />
  }

  if (view === 'kanban' && activeBoard) {
    return <Kanban board={activeBoard} onBack={handleBackToBoards} onLogout={handleLogout} />
  }

  return null
}
