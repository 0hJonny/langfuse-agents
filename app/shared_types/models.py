from pydantic import BaseModel, Field


class EvaluationResult(BaseModel):
    reasoning: str = Field(description="Почему информации достаточно или нет.")
    is_sufficient: bool = Field(description="Достаточно ли данных для ответа.")


class FactCheckResult(BaseModel):
    reasoning: str = Field(description="Комментарий фактчекера.")
    is_consistent: bool = Field(description="True, если черновик корректен.")