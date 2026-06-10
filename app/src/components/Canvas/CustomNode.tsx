import React, { memo } from 'react';
import { Handle, Position, type NodeProps } from 'reactflow';
import {
  Zap,
  Cpu,
  Share2,
  Database,
  Server,
  Globe,
  Settings,
  Wind,
  Flame,
  Layers
} from 'lucide-react';

const IconMap: Record<string, any> = {
  'api-gateway': Zap,
  'code': Cpu,
  'kafka': Share2,
  'database': Database,
  'server': Server,
  'cdn': Globe,
  'flink': Wind,
  'spark': Flame,
  'hadoop': Layers,
};

export const CustomNode = memo(({ data, selected }: NodeProps) => {
  const IconComponent = IconMap[data.type] || Settings;

  return (
    <div style={{
      background: 'rgba(15, 23, 42, 0.8)',
      backdropFilter: 'blur(12px)',
      border: `1px solid ${selected ? '#3b82f6' : 'rgba(255, 255, 255, 0.1)'}`,
      borderRadius: '12px',
      padding: '12px',
      minWidth: '160px',
      color: '#fff',
      boxShadow: selected ? '0 0 20px rgba(59, 130, 246, 0.3)' : '0 10px 30px rgba(0, 0, 0, 0.2)',
      transition: 'all 0.2s ease-in-out',
      position: 'relative'
    }}>
      <Handle type="target" position={Position.Left} style={{ background: '#3b82f6', border: 'none' }} />

      <div style={{
        position: 'absolute',
        top: 0,
        left: 0,
        right: 0,
        height: '2px',
        background: 'linear-gradient(90deg, #3b82f6, #8b5cf6)',
        opacity: 0.8,
        borderRadius: '12px 12px 0 0'
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
          <IconComponent size={18} />
        </div>
        <div>
          <h4 style={{ margin: 0, fontSize: '0.85rem', fontWeight: 600, whiteSpace: 'nowrap' }}>{data.label}</h4>
          <span style={{ fontSize: '0.55rem', color: 'rgba(255,255,255,0.4)', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
            {data.type?.replace('-', ' ')}
          </span>
        </div>
      </div>

      <div style={{
        fontSize: '0.65rem',
        color: 'rgba(255,255,255,0.6)',
        background: 'rgba(0,0,0,0.2)',
        padding: '4px 8px',
        borderRadius: '6px',
        fontFamily: 'monospace',
        overflow: 'hidden',
        textOverflow: 'ellipsis'
      }}>
        {data.id}
      </div>

      <Handle type="source" position={Position.Right} style={{ background: '#8b5cf6', border: 'none' }} />
    </div>
  );
});
