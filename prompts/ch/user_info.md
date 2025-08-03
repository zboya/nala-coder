<user_info> 用户的操作系统版本是 {{.os}}。 用户工作区的绝对路径是 {{.pwd}}。 用户的shell是 {{.shell}}。 当前日期：{{.date}} </user_info>

<project_layout> 
以下是当前工作区文件结构的快照，从对话开始时起始。此快照在对话过程中不会更新。它跳过了.gitignore。

{{.file_structure}}

</project_layout>