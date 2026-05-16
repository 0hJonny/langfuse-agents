import tiktoken

def truncate_text_by_tokens(text: str, max_tokens: int = 3000, model: str = "gpt-3.5-turbo") -> str:
    encoding = tiktoken.encoding_for_model(model)
    tokens = encoding.encode(text)
    if len(tokens) <= max_tokens:
        return text
    return encoding.decode(tokens[:max_tokens])