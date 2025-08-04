import { useState } from 'react';
import { 
  Folder, 
  FolderOpen, 
  File, 
  FileText, 
  Code, 
  Image, 
  Settings,
  ChevronRight,
  ChevronDown 
} from 'lucide-react';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

interface FileNode {
  id: string;
  name: string;
  type: 'file' | 'folder';
  children?: FileNode[];
  path: string;
  extension?: string;
}

interface FileExplorerProps {
  files: FileNode[];
  selectedFile?: string;
  onFileSelect: (file: FileNode) => void;
}

// 默认展开的文件夹（用于初始状态）
const defaultExpandedFolders = new Set(['src', '/src']);

const getFileIcon = (file: FileNode) => {
  if (file.type === 'folder') {
    return null; // 文件夹图标由展开状态决定
  }

  switch (file.extension) {
    case 'tsx':
    case 'ts':
    case 'js':
    case 'jsx':
      return <Code className="h-4 w-4 text-blue-500" />;
    case 'json':
      return <Settings className="h-4 w-4 text-yellow-500" />;
    case 'css':
    case 'scss':
    case 'less':
      return <FileText className="h-4 w-4 text-purple-500" />;
    case 'html':
      return <FileText className="h-4 w-4 text-orange-500" />;
    case 'md':
      return <FileText className="h-4 w-4 text-gray-500" />;
    case 'png':
    case 'jpg':
    case 'jpeg':
    case 'gif':
    case 'svg':
      return <Image className="h-4 w-4 text-green-500" />;
    default:
      return <File className="h-4 w-4 text-gray-400" />;
  }
};

const FileTreeNode = ({ 
  file, 
  level = 0, 
  selectedFile, 
  onFileSelect,
  expandedFolders,
  onToggleFolder 
}: {
  file: FileNode;
  level?: number;
  selectedFile?: string;
  onFileSelect: (file: FileNode) => void;
  expandedFolders: Set<string>;
  onToggleFolder: (folderId: string) => void;
}) => {
  const isExpanded = expandedFolders.has(file.id);
  const isSelected = selectedFile === file.id;

  const handleClick = () => {
    if (file.type === 'folder') {
      onToggleFolder(file.id);
    } else {
      onFileSelect(file);
    }
  };

  return (
    <div>
      <Button
        variant="ghost"
        className={cn(
          "w-full justify-start h-8 px-2 font-normal",
          isSelected && "bg-accent text-accent-foreground",
          "hover:bg-accent/50"
        )}
        style={{ paddingLeft: `${level * 16 + 8}px` }}
        onClick={handleClick}
      >
        {file.type === 'folder' && (
          <>
            {isExpanded ? (
              <ChevronDown className="h-4 w-4 mr-1" />
            ) : (
              <ChevronRight className="h-4 w-4 mr-1" />
            )}
            {isExpanded ? (
              <FolderOpen className="h-4 w-4 mr-2 text-blue-600" />
            ) : (
              <Folder className="h-4 w-4 mr-2 text-blue-600" />
            )}
          </>
        )}
        {file.type === 'file' && (
          <>
            <span className="w-5" />
            {getFileIcon(file)}
            <span className="ml-2" />
          </>
        )}
        <span className="truncate">{file.name}</span>
      </Button>

      {file.type === 'folder' && isExpanded && file.children && (
        <div>
          {file.children.map((child) => (
            <FileTreeNode
              key={child.id}
              file={child}
              level={level + 1}
              selectedFile={selectedFile}
              onFileSelect={onFileSelect}
              expandedFolders={expandedFolders}
              onToggleFolder={onToggleFolder}
            />
          ))}
        </div>
      )}
    </div>
  );
};

export const FileExplorer = ({ 
  files = [], 
  selectedFile, 
  onFileSelect 
}: FileExplorerProps) => {
  const [expandedFolders, setExpandedFolders] = useState<Set<string>>(defaultExpandedFolders);

  const handleToggleFolder = (folderId: string) => {
    setExpandedFolders(prev => {
      const newSet = new Set(prev);
      if (newSet.has(folderId)) {
        newSet.delete(folderId);
      } else {
        newSet.add(folderId);
      }
      return newSet;
    });
  };

  return (
    <div className="h-full bg-background border-r">
      <div className="p-4 border-b bg-card">
        <h2 className="text-lg font-semibold text-card-foreground">项目文件</h2>
        <p className="text-sm text-muted-foreground">浏览和编辑项目文件</p>
      </div>
      
      <ScrollArea className="flex-1">
        <div className="p-2">
          {files.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              <div className="text-sm">正在加载文件...</div>
            </div>
          ) : (
            files.map((file) => (
              <FileTreeNode
                key={file.id}
                file={file}
                selectedFile={selectedFile}
                onFileSelect={onFileSelect}
                expandedFolders={expandedFolders}
                onToggleFolder={handleToggleFolder}
              />
            ))
          )}
        </div>
      </ScrollArea>
    </div>
  );
};