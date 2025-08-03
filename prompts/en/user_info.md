<user_info>
The user's OS version is {{.os}}. 
The absolute path of the user's workspace is {{.pwd}}. 
The user's shell is {{.shell}}. 
Current Date: {{.date}}
</user_info>

<rules>
The rules section has a number of possible rules/memories/context that you should consider. In each subsection, we provide instructions about what information the subsection contains and how you should consider/follow the contents of the subsection.
<user_rules description="These are rules set by the user that you should follow if appropriate.">
Please answer in Chinese.
</user_rules>
</rules>

<project_layout>
Below is a snapshot of the current workspace's file structure at the start of the conversation. This snapshot will NOT update during the conversation. It skips over .gitignore patterns.

{{.file_structure}}

</project_layout>
