import { colors, cardStyle } from '../theme.js'

const DUE_SOON_MS = 24 * 60 * 60 * 1000

function getDueStatus(task) {
  if (!task.due_date || task.status === 'done') return null
  const diff = new Date(task.due_date).getTime() - Date.now()
  if (diff < 0) return 'overdue'
  if (diff <= DUE_SOON_MS) return 'soon'
  return null
}

export default function TaskCard({ task, onClick }) {
  function handleDragStart(e) {
    e.dataTransfer.setData('text/plain', task.id)
    e.dataTransfer.effectAllowed = 'move'
  }

  const dueStatus = getDueStatus(task)
  const accentColor =
    dueStatus === 'overdue' ? colors.danger : dueStatus === 'soon' ? colors.warning : colors.brand

  return (
    <div
      draggable
      onDragStart={handleDragStart}
      onClick={() => onClick(task)}
      style={{
        ...cardStyle,
        padding: '12px',
        marginBottom: '10px',
        cursor: 'grab',
        borderLeft: `3px solid ${accentColor}`,
      }}
    >
      <div style={{ fontSize: '14px', fontWeight: 600, marginBottom: task.description ? '4px' : 0 }}>
        {task.title}
      </div>
      {task.description && (
        <div
          style={{
            fontSize: '12px',
            color: colors.textMuted,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            display: '-webkit-box',
            WebkitLineClamp: 3,
            WebkitBoxOrient: 'vertical',
          }}
        >
          {task.description}
        </div>
      )}
      {task.due_date && (
        <div
          style={{
            fontSize: '11px',
            marginTop: '6px',
            fontWeight: 600,
            color: dueStatus === 'overdue' ? colors.danger : dueStatus === 'soon' ? colors.warning : colors.textMuted,
          }}
        >
          {dueStatus === 'overdue' ? 'Overdue · ' : dueStatus === 'soon' ? 'Due soon · ' : 'Due '}
          {new Date(task.due_date).toLocaleDateString(undefined, { timeZone: 'UTC' })}
        </div>
      )}
    </div>
  )
}
