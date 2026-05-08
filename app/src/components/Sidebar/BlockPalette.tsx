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
    <div className="sidebar">
      <h2>Architectural Blocks</h2>
      <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
        {blockTypes.map((block) => (
          <div
            key={block.type}
            className="block-item"
            onDragStart={(event) => onDragStart(event, block.type)}
            draggable
          >
            {block.icon}
            <span>{block.label}</span>
          </div>
        ))}
      </div>
    </div>
  );
};
