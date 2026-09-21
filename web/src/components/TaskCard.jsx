import { colors, cardStyle } from '../theme.js'

export default function TaskCard({ task, onClick }) {
  function handleDragStart(e) {
    e.dataTransfer.setData('text/plain', task.id)
    e.dataTransfer.effectAllowed = 'move'
  }

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
        borderLeft: `3px solid ${colors.brand}`,
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
    </div>
  )
}
