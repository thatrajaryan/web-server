import React from 'react';
import {
  Zap,
  Layers,
  Cpu,
  Share2,
  Database,
  ShieldAlert,
  Server,
  Globe
} from 'lucide-react';

import { motion } from 'framer-motion';

export const blockTypes = [
  { type: 'api_gateway', label: 'API Gateway', icon: <Zap size={18} /> },
  { type: 'load_balancer', label: 'Load Balancer', icon: <Layers size={18} /> },
  { type: 'code', label: 'Code Block', icon: <Cpu size={18} /> },
  { type: 'kafka', label: 'Kafka', icon: <Share2 size={18} /> },
  { type: 'database', label: 'Database', icon: <Database size={18} /> },
  { type: 'rate_limiter', label: 'Rate Limiter', icon: <ShieldAlert size={18} /> },
  { type: 'server', label: 'Server', icon: <Server size={18} /> },
  { type: 'cdn', label: 'CDN', icon: <Globe size={18} /> },
];

export const BlockPalette = () => {
  const onDragStart = (event: React.DragEvent, nodeType: string) => {
    event.dataTransfer.setData('application/reactflow', nodeType);
    event.dataTransfer.effectAllowed = 'move';
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
      <p style={{ 
        fontSize: '0.75rem', 
        textTransform: 'uppercase', 
        letterSpacing: '0.05em', 
        color: 'var(--text-secondary)',
        marginBottom: '4px',
        fontWeight: 600
      }}>
        Components
      </p>
      <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
        {blockTypes.map((block) => (
          <motion.div
            key={block.type}
            whileHover={{ x: 4, background: 'rgba(255,255,255,0.08)' }}
            className="block-item"
            onDragStart={(event: any) => onDragStart(event, block.type)}
            draggable
            style={{
              padding: '12px',
              borderRadius: '12px',
              border: '1px solid rgba(255,255,255,0.05)',
              background: 'rgba(255,255,255,0.03)',
              display: 'flex',
              alignItems: 'center',
              gap: '12px',
              cursor: 'grab'
            }}
          >
            <div style={{ color: '#3b82f6' }}>{block.icon}</div>
            <span style={{ fontSize: '0.9rem', fontWeight: 500 }}>{block.label}</span>
          </motion.div>
        ))}
      </div>
    </div>
  );
};
