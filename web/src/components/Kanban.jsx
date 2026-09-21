import { useEffect, useState } from 'react'
import { api } from '../api.js'
import { colors, cardStyle, fontFamily, pageStyle } from '../theme.js'
import Button from './common/Button.jsx'
import TaskCard from './TaskCard.jsx'
import TaskForm from './TaskForm.jsx'

const COLUMNS = [
  { status: 'todo', title: 'To do' },
  { status: 'in_progress', title: 'In progress' },
  { status: 'done', title: 'Done' },
]

export default function Kanban({ board, onBack, onLogout }) {
  const [tasks, setTasks] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [formOpen, setFormOpen] = useState(false)
  const [editingTask, setEditingTask] = useState(null)

  useEffect(() => {
    refresh()
  }, [board.id])

  async function refresh() {
    setLoading(true)
    try {
      const data = await api.listTasks(board.id)
      setTasks(data || [])
    } catch (err) {
      setError(err.message || 'Failed to load tasks')
    } finally {
      setLoading(false)
    }
  }

  async function handleDrop(taskId, newStatus) {
    const task = tasks.find((t) => t.id === taskId)
    if (!task || task.status === newStatus) return

    const previous = tasks
    setTasks((prev) => prev.map((t) => (t.id === taskId ? { ...t, status: newStatus } : t)))
    try {
      await api.updateTask(taskId, { status: newStatus })
    } catch (err) {
      setTasks(previous)
      setError(err.message || 'Failed to move task')
    }
  }

  function openCreate() {
    setEditingTask(null)
    setFormOpen(true)
  }

  function openEdit(task) {
    setEditingTask(task)
    setFormOpen(true)
  }

  function closeForm() {
    setFormOpen(false)
    setEditingTask(null)
  }

  async function handleSave(values) {
    if (editingTask) {
      const updated = await api.updateTask(editingTask.id, values)
      setTasks((prev) => prev.map((t) => (t.id === updated.id ? updated : t)))
    } else {
      const created = await api.createTask({ ...values, board_id: board.id })
      setTasks((prev) => [...prev, created])
    }
    closeForm()
  }

  async function handleDelete(task) {
    if (!confirm(`Delete task "${task.title}"?`)) return
    try {
      await api.deleteTask(task.id)
      setTasks((prev) => prev.filter((t) => t.id !== task.id))
      closeForm()
    } catch (err) {
      setError(err.message || 'Failed to delete task')
    }
  }

  return (
    <div style={{ ...pageStyle, padding: '24px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h1 style={{ margin: 0, fontSize: '22px', color: colors.brand, fontFamily }}>Todo App</h1>
        <button style={linkBtn} onClick={onLogout}>
          Log out
        </button>
      </div>

      <div style={{ display: 'flex', alignItems: 'center', gap: '12px', margin: '20px 0 4px' }}>
        <button style={linkBtn} onClick={onBack}>
          &larr; Boards
        </button>
      </div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
        <div>
          <h2 style={{ margin: '0 0 4px', fontSize: '20px' }}>{board.name}</h2>
          {board.summary && <p style={{ margin: 0, fontSize: '13px', color: colors.textMuted }}>{board.summary}</p>}
        </div>
        <Button onClick={openCreate}>+ New task</Button>
      </div>

      {loading && <p style={{ color: colors.textMuted }}>Loading…</p>}
      {error && <p style={{ color: colors.danger }}>{error}</p>}

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', gap: '16px' }}>
        {COLUMNS.map((col) => (
          <Column
            key={col.status}
            title={col.title}
            status={col.status}
            tasks={tasks.filter((t) => t.status === col.status)}
            onDropTask={handleDrop}
            onTaskClick={openEdit}
          />
        ))}
      </div>

      {formOpen && (
        <TaskForm
          task={editingTask}
          onSave={handleSave}
          onDelete={editingTask ? handleDelete : undefined}
          onClose={closeForm}
        />
      )}
    </div>
  )
}

function Column({ title, status, tasks, onDropTask, onTaskClick }) {
  const [dragOver, setDragOver] = useState(false)

  function handleDragOver(e) {
    e.preventDefault()
    e.dataTransfer.dropEffect = 'move'
    if (!dragOver) setDragOver(true)
  }

  function handleDrop(e) {
    e.preventDefault()
    setDragOver(false)
    const taskId = e.dataTransfer.getData('text/plain')
    if (taskId) onDropTask(taskId, status)
  }

  return (
    <div
      onDragOver={handleDragOver}
      onDragLeave={() => setDragOver(false)}
      onDrop={handleDrop}
      style={{
        ...cardStyle,
        background: dragOver ? colors.brandLight : colors.surface,
        padding: '12px',
        minHeight: '300px',
        transition: 'background 0.1s ease',
      }}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '10px' }}>
        <h3 style={{ margin: 0, fontSize: '13px', textTransform: 'uppercase', letterSpacing: '0.04em', color: colors.textMuted }}>
          {title}
        </h3>
        <span style={{ fontSize: '12px', color: colors.textMuted }}>{tasks.length}</span>
      </div>
      {tasks.map((task) => (
        <TaskCard key={task.id} task={task} onClick={onTaskClick} />
      ))}
    </div>
  )
}

const linkBtn = {
  background: 'none',
  border: 'none',
  padding: 0,
  color: colors.brand,
  cursor: 'pointer',
  fontSize: '13px',
  fontWeight: 600,
}
