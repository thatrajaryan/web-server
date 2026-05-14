import React from 'react';
import {
  Zap,
  Cpu,
  Share2,
  Database,
  Server,
  Globe,
  Settings
} from 'lucide-react';
import { motion } from 'framer-motion';

export const blockTypes = [
  { type: 'api-gateway', label: 'API Gateway' },
  { type: 'kafka', label: 'Kafka' },
  { type: 'database', label: 'Database' },
  { type: 'server', label: 'Server' },
  { type: 'cdn', label: 'CDN' },
];

const IconMap: Record<string, any> = {
  'api-gateway': Zap,
  'kafka': Share2,
  'database': Database,
  'server': Server,
  'cdn': Globe,
};

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
        {blockTypes.map((block) => {
          const IconComponent = IconMap[block.type] || Settings;
          return (
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
              <div style={{ color: '#3b82f6', display: 'flex' }}>
                <IconComponent size={18} />
              </div>
              <span style={{ fontSize: '0.9rem', fontWeight: 500 }}>{block.label}</span>
            </motion.div>
          );
        })}
      </div>
    </div>
  );
};
