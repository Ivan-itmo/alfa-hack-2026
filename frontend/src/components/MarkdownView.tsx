import React from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';

interface MarkdownViewProps {
  content?: string | null;
  className?: string;
  style?: React.CSSProperties;
}

export const MarkdownView: React.FC<MarkdownViewProps> = ({ content, className = '', style }) => {
  if (!content) return null;

  return (
    <div className={`markdown-view ${className}`} style={style}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          h1: ({ node, ...props }) => <h3 className="markdown-h3" {...props} />,
          h2: ({ node, ...props }) => <h3 className="markdown-h3" {...props} />,
          h3: ({ node, ...props }) => <h3 className="markdown-h3" {...props} />,
          h4: ({ node, ...props }) => <h4 className="markdown-h4" {...props} />,
          p: ({ node, ...props }) => <p className="markdown-p" {...props} />,
          ul: ({ node, ...props }) => <ul className="markdown-ul" {...props} />,
          ol: ({ node, ...props }) => <ol className="markdown-ol" {...props} />,
          li: ({ node, ...props }) => <li className="markdown-li" {...props} />,
          strong: ({ node, ...props }) => <strong className="markdown-strong" {...props} />,
          code: ({ node, ...props }) => <code className="markdown-code" {...props} />,
          table: ({ node, ...props }) => (
            <div className="markdown-table-wrapper">
              <table className="markdown-table" {...props} />
            </div>
          ),
          th: ({ node, ...props }) => <th className="markdown-th" {...props} />,
          td: ({ node, ...props }) => <td className="markdown-td" {...props} />,
          blockquote: ({ node, ...props }) => <blockquote className="markdown-blockquote" {...props} />,
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  );
};

export default MarkdownView;
