import React, { memo } from 'react';
import { Handle, Position } from 'reactflow';

export const CustomNode = memo(({ data }: any) => {
  return (
    <div style={{
      background: 'rgba(30, 41, 59, 0.7)',
      backdropFilter: 'blur(16px)',
      border: '1px solid rgba(255, 255, 255, 0.1)',
      borderRadius: '8px',
      padding: '10px',
      minWidth: '120px',
      color: '#fff',
      boxShadow: '0 5px 15px rgba(0, 0, 0, 0.5)',
      position: 'relative',
      overflow: 'hidden'
    }}>
      {/* Decorative gradient overlay */}
      <div style={{
        position: 'absolute',
        top: 0,
        left: 0,
        right: 0,
        height: '2px',
        background: 'linear-gradient(90deg, #3b82f6, #8b5cf6)',
        opacity: 0.8
      }} />

      <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '8px' }}>
        <div style={{ 
          background: 'rgba(59, 130, 246, 0.1)', 
          padding: '6px', 
          borderRadius: '6px',
          color: '#3b82f6',
          display: 'flex',
          transform: 'scale(0.8)'
        }}>
          {data.icon}
        </div>
        <div>
          <h4 style={{ margin: 0, fontSize: '0.85rem', fontWeight: 600, whiteSpace: 'nowrap' }}>{data.label}</h4>
          <span style={{ fontSize: '0.55rem', color: 'rgba(255,255,255,0.4)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
            {data.type?.replace('_', ' ')}
          </span>
        </div>
      </div>

      <div style={{ 
        fontSize: '0.65rem', 
        color: 'rgba(255,255,255,0.6)', 
        background: 'rgba(0,0,0,0.2)',
        padding: '4px 8px',
        borderRadius: '4px',
        fontFamily: 'monospace',
        overflow: 'hidden',
        textOverflow: 'ellipsis'
      }}>
        {data.id}
      </div>

      <Handle
        type="target"
        position={Position.Top}
        style={{ background: '#3b82f6', border: '1px solid #1e293b', width: '6px', height: '6px' }}
      />
      <Handle
        type="source"
        position={Position.Bottom}
        style={{ background: '#3b82f6', border: '1px solid #1e293b', width: '6px', height: '6px' }}
      />
    </div>
  );
});
